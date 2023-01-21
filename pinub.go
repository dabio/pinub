package pinub

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"dab.io/pinub/internal/cookies"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed templates/*.html
var tpls embed.FS

type key int

const (
	layoutTpl = "templates/_layout.html"

	cookieName = "keks"
	// 30 days * 24 hours * 60 minutes * 60 seconds
	cookieMaxAge = 30 * 24 * 60 * 60

	// context key for storing user object
	userContextKey key = 0
)

type App struct {
	ListenAddress string
	DSN           string
	SecretKey     []byte

	userService *UserService
}

func (a *App) Start() {
	db, err := sql.Open("sqlite", a.DSN)
	if err != nil {
		slog.Error("cannot open database", err, "dsn", a.DSN)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("cannot ping database", err, "dsn", a.DSN)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		slog.Error("cannot run migrations", err, "dsn", a.DSN)
	}
	a.userService = &UserService{DB: db}

	m := http.NewServeMux()
	m.HandleFunc("/", private(a.index()))
	m.HandleFunc("/home", a.home())
	m.HandleFunc("/signin", a.signin())
	m.HandleFunc("/register", a.register())
	m.HandleFunc("/profile", private(a.profile()))
	m.HandleFunc("/signout", private(a.signout()))
	m.HandleFunc("/_healthz", healthz(db))

	slog.Info("starting", "address", a.ListenAddress)
	slog.Error("server failed", http.ListenAndServe(a.ListenAddress, logreq(a.auth(m))))
}

func (a *App) index() http.HandlerFunc {
	tpl, _ := template.New("index.html").Funcs(template.FuncMap{
		// remove http and https scheme from urls
		"lremove": func(prefix, s string) string {
			return strings.TrimPrefix(s, prefix)
		},
		// timesince for createdAt
		"timesince": func(createdAt *time.Time) string {
			diff := time.Since(*createdAt)
			if diff.Hours() > 24 {
				return createdAt.Format("02.01.06 15:04:05")
			}
			if diff.Hours() > 1 {
				return fmt.Sprintf("%.0fh ago", diff.Hours())
			}
			if diff.Minutes() > 1 {
				return fmt.Sprintf("%.0fm ago", diff.Minutes())
			}

			return fmt.Sprintf("%.0fs ago", diff.Seconds())
		},
		"format": func(at *time.Time, format string) string {
			return at.Format(format)
		},
	}).ParseFS(tpls, "templates/index.html", layoutTpl)

	ignoredFiles := map[string]bool{
		"apple-touch-icon-152x152-precomposed.png": true,
		"apple-touch-icon-152x152.png":             true,
		"apple-touch-icon-120x120-precomposed.png": true,
		"apple-touch-icon-120x120.png":             true,
		"apple-touch-icon-precomposed.png":         true,
		"apple-touch-icon.png":                     true,
		"favicon.ico":                              true,
	}
	var data struct {
		User  *User
		Links []Link
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*User)

		rawLink := strings.TrimSpace(r.URL.String())[1:]
		// show list of links
		if len(rawLink) == 0 {
			links, err := a.userService.Links(r.Context(), user)
			if err != nil {
				http.Error(w, "cannot get links from database", http.StatusBadRequest)
			}

			data.User = user
			data.Links = links

			render(w, tpl, data)
			return
		}

		// a new link was provided by the user
		if _, ok := ignoredFiles[rawLink]; ok {
			http.NotFound(w, r)
			return
		}

		// fix https:/example.com - single :/ after scheme
		if strings.Contains(rawLink, ":/") && !strings.Contains(rawLink, "://") {
			rawLink = strings.Join(strings.SplitN(rawLink, ":/", 2), "://")
		}

		if !strings.HasPrefix(rawLink, "http") {
			rawLink = "http://" + rawLink
		}
		if len(rawLink) < 10 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		url, err := url.Parse(rawLink)
		if err != nil {
			http.Error(w, "link is not valid ", http.StatusBadRequest)
		}

		link := &Link{
			URL: url.String(),
		}

		if err := a.userService.Addlink(r.Context(), user, link); err != nil {
			http.Error(w, "cannot add link to user", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (a *App) home() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/home.html", layoutTpl)

	return func(w http.ResponseWriter, r *http.Request) {
		render(w, tpl, nil)
	}
}

func (a *App) signin() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/signin.html", layoutTpl)

	return func(w http.ResponseWriter, r *http.Request) {
		// show form
		if r.Method == http.MethodGet {
			render(w, tpl, nil)
			return
		}

		// check for valid email
		mail, err := mail.ParseAddress(strings.TrimSpace(r.FormValue("email")))
		if err != nil {
			http.Error(w, "email address is not valid", http.StatusBadRequest)
			return
		}

		// check if user is already present
		user, err := a.userService.ByEmail(r.Context(), mail.Address)
		if err != nil {
			http.Error(w, "email is unknown", http.StatusBadRequest)
			return
		}

		pass := strings.TrimSpace(r.FormValue("password"))
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pass)); err != nil {
			http.Error(w, "password not valid", http.StatusBadRequest)
			return
		}

		// create token
		if err := a.userService.CreateToken(r.Context(), user); err != nil {
			http.Error(w, "cannot create token", http.StatusBadRequest)
			return
		}

		// set the cookie
		cookie := http.Cookie{
			Name:     cookieName,
			Value:    user.Token,
			Path:     "/",
			MaxAge:   cookieMaxAge,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteStrictMode,
		}
		if err := cookies.WriteEncrypted(w, cookie, a.SecretKey); err != nil {
			http.Error(w, "cannot save cookie", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (a *App) signout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(cookieName); err == nil { // if NO error
			// remove cookie
			cookie.MaxAge = -1
			cookie.Path = "/"
			http.SetCookie(w, cookie)
		}
		http.Redirect(w, r, "/home", http.StatusSeeOther)
	}
}

func (a *App) register() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/register.html", layoutTpl)

	return func(w http.ResponseWriter, r *http.Request) {
		// show form
		if r.Method == http.MethodGet {
			render(w, tpl, nil)
			return
		}

		// check for password equals second password
		password := strings.TrimSpace(r.FormValue("password"))
		passrepa := strings.TrimSpace(r.FormValue("passrepa"))
		if len(password) < 3 {
			http.Error(w, "password is too short", http.StatusBadRequest)
			return
		}
		if password != passrepa {
			http.Error(w, "passwords do not match", http.StatusBadRequest)
			return
		}
		// check for valid email
		mail, err := mail.ParseAddress(strings.TrimSpace(r.FormValue("email")))
		if err != nil {
			http.Error(w, "email address is not valid", http.StatusBadRequest)
			return
		}

		// check if user is already present
		user, err := a.userService.ByEmail(r.Context(), mail.Address)
		if user != nil {
			http.Error(w, "email already in database", http.StatusBadRequest)
			return
		}

		// hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		user = &User{
			Email:    mail.Address,
			Password: string(hash),
		}

		// create user
		if err := a.userService.CreateUser(r.Context(), user); err != nil {
			http.Error(w, "cannot create user", http.StatusBadRequest)
			return
		}

		// create token
		if err := a.userService.CreateToken(r.Context(), user); err != nil {
			http.Error(w, "cannot create token", http.StatusBadRequest)
			return
		}

		// set the cookie
		cookie := http.Cookie{
			Name:     cookieName,
			Value:    user.Token,
			Path:     "/",
			MaxAge:   cookieMaxAge,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
		if err := cookies.WriteEncrypted(w, cookie, a.SecretKey); err != nil {
			http.Error(w, "cannot save cookie", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (a *App) profile() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/profile.html", layoutTpl)

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*User)

		if r.Method == http.MethodGet {
			render(w, tpl, user)
			return
		}

		// check for valid password
		pass := strings.TrimSpace(r.FormValue("pass"))
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pass)); err != nil {
			http.Error(w, "password not valid", http.StatusBadRequest)
			return
		}

		// update email
		mail, err := mail.ParseAddress(strings.TrimSpace(r.FormValue("email")))
		if err != nil {
			http.Error(w, "cannot parse email", http.StatusBadRequest)
			return
		}
		if err := a.userService.UpdateEmail(r.Context(), user, mail.Address); err != nil {
			http.Error(w, "cannot update email", http.StatusBadRequest)
			return
		}

		// update password
		newpass := strings.TrimSpace(r.FormValue("newpass"))
		if len(newpass) > 0 {
			// hash password
			hash, err := bcrypt.GenerateFromPassword([]byte(newpass), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "cannot hash password", http.StatusBadRequest)
				return
			}

			if err := a.userService.UpdatePassword(r.Context(), user, string(hash)); err != nil {
				http.Error(w, "cannot update password", http.StatusBadRequest)
				return
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

func healthz(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, fmt.Sprintf("Database error: %s", err), http.StatusFailedDependency)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (a *App) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token, err := cookies.ReadEncrypted(r, cookieName, a.SecretKey); err == nil { // if NO error
			if user, err := a.userService.ByToken(r.Context(), token); err == nil { // of NO error
				// extend the cookie
				cookie := http.Cookie{
					Name:     cookieName,
					Value:    user.Token,
					Path:     "/",
					MaxAge:   cookieMaxAge,
					HttpOnly: true,
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
				}
				if err := cookies.WriteEncrypted(w, cookie, a.SecretKey); err != nil {
					slog.Error("cannot save cookie", err)
				}

				if err := a.userService.UpdateToken(r.Context(), token); err != nil {
					slog.Error("couldn't update token", err)
				}
				// extend token in db

				ctx := context.WithValue(r.Context(), userContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// cookie present, but no user in database found. remove cookie
			if cookie, err := r.Cookie(cookieName); err == nil { // if NO error
				// remove cookie
				cookie.MaxAge = -1
				cookie.Path = "/"
				http.SetCookie(w, cookie)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// func public(next http.HandlerFunc) http.HandlerFunc {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if _, ok := r.Context().Value(userContextKey).(*User); ok {
// 			http.Redirect(w, r, "/", http.StatusSeeOther)
// 		} else {
// 			next(w, r)
// 		}
// 	})
// }

func private(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(userContextKey).(*User); !ok {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
		} else {
			next(w, r)
		}
	})
}

func render(w http.ResponseWriter, tpl *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf8")

	if err := tpl.Execute(w, data); err != nil {
		slog.Error("error executing template", err)
	}
}

// statusRecorder is used to overwrite the WriteHeader function in
// http.ResponseWriter. WriteHeader will save the status code of a request
// in this struct's status field. Later on we can use this field to log the
// request - in logreq().
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

func logreq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to statusRecorder to get the status in our log.
		// Set default code to 200.
		rec := &statusRecorder{w, http.StatusOK}

		next.ServeHTTP(rec, r)

		defer slog.Info(
			"access",
			"remote_addr", r.RemoteAddr,
			"duration", time.Since(start).String(),
			"method", r.Method,
			"uri", r.URL.String(),
			"proto", r.Proto,
			"status", rec.status,
		)
	})
}
