package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pinub/mux/v3"
)

type server struct {
	ctx  context.Context
	db   client
	tpl  *tpl
	user string
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
	m.Post("/signin", s.public(s.todo()))
	m.Get("/register", s.public(s.showRegister()))
	m.Post("/register", s.public(s.register()))
	// private
	m.Get("/profile", s.private(s.todo()))
	m.Post("/profile", s.private(s.todo()))
	m.Get("/signout", s.private(s.todo()))
	m.NotFound = s.todo()

	h := s.auth(m)

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), h))
}

func (s *server) auth(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("auth")
		h.ServeHTTP(w, r)
	}
}

func (s *server) private(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("private")
		h(w, r)
	}
}

func (s *server) public(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("public")
		h(w, r)
	}
}

func (s *server) showRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, "register.html", struct{ User *user }{User: &user{}})
	}
}

func (s *server) register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := &user{
			Email:    strings.TrimSpace(r.FormValue("email")),
			Password: strings.TrimSpace(r.FormValue("password")),
		}
		if err := s.db.UserService().CreateUser(s.ctx, u); err != nil {
			s.tpl.render(w, "register.html", struct{ User *user }{User: u})
			return
		}
		if err := s.db.UserService().UserAddToken(s.ctx, u); err != nil {
			s.tpl.render(w, "register.html", struct{ User *user }{User: u})
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *server) showSignin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.tpl.render(w, "signin.html", nil)
	}
}

func (s *server) todo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello`))
	}
}
