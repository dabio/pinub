package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pinub/mux/v3"
	"golang.org/x/crypto/bcrypt"
)

const cookieName = "keks"
const cookieDays = 14

type server struct {
	ctx  context.Context
	db   client
	tpl  *tpl
	user *user
}

// Serve will configure the routes and start the http server.
func Serve() {
	s := &server{
		ctx: context.Background(),
		tpl: newTpl("templates/"),
		db:  newPg(os.Getenv("DATABASE_URL")),
	}

	m := mux.New()
	// public
	m.Get("/signin", s.public(s.showSignin()))
	m.Post("/signin", s.public(s.signin()))
	m.Get("/register", s.public(s.showRegister()))
	m.Post("/register", s.public(s.register()))
	// private
	m.Get("/profile", s.private(s.todo()))
	m.Post("/profile", s.private(s.todo()))
	m.Get("/signout", s.private(s.signout()))
	m.NotFound = s.todo()

	// middlewares
	h := s.auth(m)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), h))
}

// middlewares

func (s *server) auth(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(cookieName); err == nil {
			if u, err := s.db.UserService().UserByToken(s.ctx, c.Value); err == nil {
				s.user = u
				if !strings.HasPrefix(r.URL.String(), "/signout") {
					refreshCookie(w, c)
				}
				go s.db.UserService().UserRefreshToken(s.ctx, u)
			} else {
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

func (s *server) showRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, "register.html", map[string]interface{}{
			"User": &user{},
		})
	}
}

func (s *server) register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := &user{
			Email:    strings.TrimSpace(r.FormValue("email")),
			Password: strings.TrimSpace(r.FormValue("password")),
		}
		data := map[string]interface{}{
			"User": u,
		}
		// check for valid fields
		if err := u.Validate(); err != nil {
			data["Error"] = err
			s.tpl.render(w, "register.html", data)
			return
		}
		// check for password confirmation
		if u.Password != strings.TrimSpace(r.FormValue("password_confirm")) {
			data["Error"] = fmt.Errorf("passwords do not match")
			s.tpl.render(w, "register.html", data)
			return
		}
		// check if user is already in database
		if _, err := s.db.UserService().UserByEmail(s.ctx, u.Email); err != sql.ErrNoRows {
			data["Error"] = fmt.Errorf("user exists already")
			s.tpl.render(w, "register.html", data)
			return
		}

		u.Password, _ = hashPassword(u.Password)
		if err := s.db.UserService().CreateUser(s.ctx, u); err != nil {
			data["Error"] = fmt.Errorf("cannot persist user to database")
			s.tpl.render(w, "register.html", data)
			return
		}

		if err := s.db.UserService().UserAddToken(s.ctx, u); err != nil {
			log.Print("Cannot save a new token for newly created user")
		}

		createCookie(w, cookieName, u.Token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *server) showSignin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, "signin.html", map[string]interface{}{
			"User": &user{},
		})
	}
}

func (s *server) signin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := strings.TrimSpace(r.FormValue("email"))
		passw := strings.TrimSpace(r.FormValue("password"))

		u, err := s.db.UserService().UserByEmail(s.ctx, email)
		data := map[string]interface{}{
			"User": u,
		}
		if err != nil {
			data["Error"] = "email is unknown"
			s.tpl.render(w, "signin.html", data)
			return
		}

		if !isValidPassword(u.Password, passw) {
			data["Error"] = "cannot sign in user"
			s.tpl.render(w, "signin.html", data)
			return
		}

		if err := s.db.UserService().UserAddToken(s.ctx, u); err != nil {
			log.Print("Cannot save a new token for signed in user")
		}

		createCookie(w, cookieName, u.Token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *server) signout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(cookieName); err == nil {
			deleteCookie(w, cookie)
		}

		s.user = nil
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
	}
}

func (s *server) todo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello`))
	}
}

// private handlers

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

func isValidPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	return string(hash), err
}
