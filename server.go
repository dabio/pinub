package main

import (
	"log"
	"net/http"
	"os"

	"github.com/pinub/mux/v3"
)

type server struct {
	db   client
	tpl  *tpl
	user string
}

// Serve will configure the routes and start the http server.
func Serve() {
	s := &server{
		tpl: newTpl("templates/"),
		db:  newPg(os.Getenv("DATABASE_URL")),
	}

	m := mux.New()
	// public
	m.Get("/signin", s.public(s.showSignin()))
	m.Post("/signin", s.public(s.todo()))
	m.Get("/register", s.public(s.showRegister()))
	m.Post("/register", s.public(s.todo()))
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
		s.tpl.render(w, "register.html", nil)
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
