package pinub

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pinub/mux/v3"
	_ "modernc.org/sqlite"
)

const (
	layoutTpl   = "templates/_layout.html"
	indexTpl    = "templates/index.html"
	homeTpl     = "templates/home.html"
	signinTpl   = "templates/signin.html"
	registerTpl = "templates/register.html"
	profileTpl  = "templates/profile.html"

	cookieName = "keks"
	cookieDays = 30
)

//go:embed templates/*.html
var tpls embed.FS

type App struct {
	Address    string
	StaticBase string
	DB         *sql.DB

	linkService *LinkService
	userService *UserService
}

func (a *App) Start() {
	a.linkService = &LinkService{DB: a.DB}
	a.userService = &UserService{DB: a.DB}

	m := mux.New()
	m.Get("/", a.index())
	m.Get("/home", a.home())
	m.Get("/signin", a.signin())
	m.Post("/signin", a.signin())
	m.Get("/signout", a.signout())
	m.Get("/register", a.register())
	m.Get("/profile", a.profile())
	m.Get("/_healthz", a.healthz())

	m.Use(logreq)

	log.Printf("Starting app on %s", a.Address)
	log.Fatal(http.ListenAndServe(a.Address, logreq(m)))
}

func (a *App) index() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, indexTpl, layoutTpl)
	const timeout = 1 * time.Second

	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// ToDo: set correct user identifier
		links, err := a.linkService.Links(ctx, "1")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		render(rw, tpl, links)
	}
}

func (a *App) home() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, homeTpl, layoutTpl)

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) signin() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, signinTpl, layoutTpl)
	const timeout = 1 * time.Second

	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		email := strings.TrimSpace(r.FormValue("email"))
		password := strings.TrimSpace(r.FormValue("password"))

		if email != "" || password != "" {
			user, err := a.userService.LoginUser(ctx, email, password)
			if err != nil {
				// ToDo: inform about error
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			createCookie(rw, user.Token)
			http.Redirect(rw, r, "/", http.StatusSeeOther)
		}

		render(rw, tpl, nil)
	}
}

func (a *App) signout() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(cookieName); err == nil {
			deleteCookie(rw, c)
		}

		http.Redirect(rw, r, "/signin", http.StatusSeeOther)
	}
}

func (a *App) register() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, registerTpl, layoutTpl)

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) profile() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, profileTpl, layoutTpl)

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) healthz() http.HandlerFunc {
	const timeout = 1 * time.Second

	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		if err := a.DB.PingContext(ctx); err != nil {
			http.Error(rw, fmt.Sprintf("db down: %v", err), http.StatusFailedDependency)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func createCookie(rw http.ResponseWriter, value string) {
	c := &http.Cookie{
		Name:    cookieName,
		Value:   value,
		Path:    "/",
		Expires: time.Now().Add(cookieDays * time.Hour * 24),
	}

	http.SetCookie(rw, c)
}

func deleteCookie(rw http.ResponseWriter, c *http.Cookie) {
	c.Path = "/"
	c.MaxAge = -1
	http.SetCookie(rw, c)
}

func render(rw http.ResponseWriter, tpl *template.Template, data interface{}) {
	rw.Header().Set("Content-Type", "text/html; charset=utf8")

	if err := tpl.Execute(rw, data); err != nil {
		log.Fatal(err)
	}
}

func logreq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(rw, r)

		defer log.Printf(
			"%v %v %v %v",
			time.Since(start),
			r.Method,
			r.URL.String(),
			r.Proto,
		)
	})
}
