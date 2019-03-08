package handler

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dabio/pinub/auth"
	"github.com/dabio/pinub/postgres"
	"github.com/pinub/mux/v3"
)

const (
	cookieName = "keks"
	cookieDays = 14

	minPassLen = 4

	profileTpl  = "profile.html"
	registerTpl = "register.html"
	signinTpl   = "signin.html"
)

type server struct {
	tpl  *tpl
	auth *auth.Client
	user *auth.User
}

// Serve will configure the routes and start the http server.
func Serve() {
	db := postgres.NewClient(os.Getenv("DATABASE_URL"))

	s := &server{
		auth: auth.NewClient(auth.Config{MinPassLen: minPassLen}, db),
		tpl:  newTpl("templates/"),
	}

	m := mux.New()
	// public
	m.Get("/signin", s.public(s.showSignin()))
	m.Post("/signin", s.public(s.signin()))
	m.Get("/register", s.public(s.showRegister()))
	m.Post("/register", s.public(s.register()))
	// private
	m.Get("/profile", s.private(s.showProfile()))
	m.Post("/profile", s.private(s.profile()))
	m.Get("/signout", s.private(s.signout()))
	m.NotFound = s.todo()

	// middlewares
	h := s.authenticate(m)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), h))
}

// middlewares

func (s *server) authenticate(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(cookieName); err == nil {
			if u, err := s.auth.AccountInfo(c.Value); err == nil {
				s.user = u
				if !strings.HasPrefix(r.URL.String(), "/signout") {
					refreshCookie(w, c)
				}
				// go s.db.UserService().RefreshToken(s.ctx, u.ID())
			} else {
				log.Print(err)
				deleteCookie(w, c)
			}
		}
		h.ServeHTTP(w, r)
		// set to nil for next request
		s.user = nil
	}
}

func (s *server) private(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.user == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			h(w, r)
		}
	}
}

func (s *server) public(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.user != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			h(w, r)
		}
	}
}

// public handlers

type register struct {
	User  *auth.User
	Error error
}

func (s *server) showRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, registerTpl, &register{User: &auth.User{}})
	}
}

func (s *server) register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.FormValue("email"))
		passw := strings.TrimSpace(r.FormValue("password"))

		data := &register{User: &auth.User{Email: email}}
		u, err := s.auth.SignupNewUser(email, passw)
		if err != nil {
			data.Error = err
			s.tpl.render(w, registerTpl, data)
			return
		}

		createCookie(w, cookieName, u.Token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

type signin struct {
	User  *auth.User
	Error error
}

func (s *server) showSignin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, signinTpl, &signin{User: &auth.User{}})
	}
}

func (s *server) signin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.FormValue("email"))
		passw := strings.TrimSpace(r.FormValue("password"))

		data := &signin{User: &auth.User{Email: email}}

		user, err := s.auth.VerifyPassword(email, passw)
		if err != nil {
			data.Error = err
			s.tpl.render(w, signinTpl, data)
			return
		}

		createCookie(w, cookieName, user.Token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// private handlers

func (s *server) signout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(cookieName); err == nil {
			deleteCookie(w, cookie)
		}

		s.user = nil
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}
}

type profile struct {
	User  *auth.User
	Error error
}

func (s *server) showProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, profileTpl, &profile{User: s.user})
	}
}

func (s *server) profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.FormValue("email"))
		curpw := strings.TrimSpace(r.FormValue("password"))
		newpw := strings.TrimSpace(r.FormValue("new_password"))

		data := &profile{User: s.user}
		switch {
		case len(email) > 0 && email != s.user.Email:
			if err := s.auth.ChangeEmail(s.user.Token, email); err != nil {
				data.Error = err
				s.tpl.render(w, profileTpl, data)
				return
			}
		case len(newpw) > 0:
			if _, err := s.auth.VerifyPassword(s.user.Email, curpw); err != nil {
				data.Error = err
				s.tpl.render(w, profileTpl, data)
				return
			}
			if err := s.auth.ChangePassword(s.user.Token, newpw); err != nil {
				data.Error = err
				s.tpl.render(w, profileTpl, data)
				return
			}
		}

		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

func (s *server) todo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello`))
	}
}

// helpers

func createCookie(w http.ResponseWriter, name, value string) {
	cookie := &http.Cookie{
		Name:    name,
		Value:   value,
		Path:    "/",
		Expires: time.Now().Add(cookieDays * time.Hour * 24),
	}
	http.SetCookie(w, cookie)
}

func refreshCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(cookieDays * time.Hour * 24)
	http.SetCookie(w, cookie)
}

func deleteCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Path = "/"
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}
