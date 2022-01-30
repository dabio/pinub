package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/pinub/mux/v3"
)

//go:embed templates/*.html
var tpls embed.FS

type App struct {
	Port       string
	StaticBase string
}

func (a *App) Start() {
	m := mux.New()
	m.Get("/", a.index())
	m.Get("/signin", a.signin())
	m.Get("/register", a.register())
	m.Get("/profile", a.profile())

	log.Printf("Starting app on %s", a.Port)
	log.Fatal(http.ListenAndServe(a.Port, m))
}

func (a *App) index() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/index.html", "templates/_layout.html")

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) signin() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/signin.html", "templates/_layout.html")

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) register() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/register.html", "templates/_layout.html")

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func (a *App) profile() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, "templates/profile.html", "templates/_layout.html")

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
	}
}

func render(rw http.ResponseWriter, tpl *template.Template, data interface{}) {
	rw.Header().Set("Content-Type", "text/html; charset=utf8")

	if err := tpl.Execute(rw, data); err != nil {
		log.Fatal(err)
	}
}

func env(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return defaultValue
}

func main() {
	server := App{
		Port:       ":" + env("PORT", "8080"),
		StaticBase: env("STATIC_BASE", "/static"),
	}
	server.Start()
}
