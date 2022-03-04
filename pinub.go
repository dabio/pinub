package pinub

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/pinub/mux/v3"
	_ "modernc.org/sqlite"
)

const (
	layoutTpl   = "templates/_layout.html"
	indexTpl    = "templates/index.html"
	signinTpl   = "templates/signin.html"
	registerTpl = "templates/register.html"
	profileTpl  = "templates/profile.html"
)

//go:embed templates/*.html
var tpls embed.FS

type App struct {
	Port       string
	StaticBase string
	DB         *sql.DB

	linkService *LinkService
}

func (a *App) Start() {
	a.linkService = &LinkService{DB: a.DB}

	m := mux.New()
	m.Get("/", a.index())
	m.Get("/signin", a.signin())
	m.Get("/register", a.register())
	m.Get("/profile", a.profile())
	m.Get("/healthz", a.healthz())

	log.Printf("Starting app on %s", a.Port)
	log.Fatal(http.ListenAndServe(a.Port, logreq(m)))
}

func (a *App) index() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, indexTpl, layoutTpl)
	const timeout = 1

	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Second)
		defer cancel()

		links, err := a.linkService.Links(ctx, "1")
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		render(rw, tpl, links)
	}
}

func (a *App) signin() http.HandlerFunc {
	tpl, _ := template.ParseFS(tpls, signinTpl, layoutTpl)

	return func(rw http.ResponseWriter, r *http.Request) {
		render(rw, tpl, nil)
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
	const timeout = 1

	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout*time.Second)
		defer cancel()

		if err := a.DB.PingContext(ctx); err != nil {
			http.Error(rw, fmt.Sprintf("db down: %v", err), http.StatusFailedDependency)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func render(rw http.ResponseWriter, tpl *template.Template, data interface{}) {
	rw.Header().Set("Content-Type", "text/html; charset=utf8")

	if err := tpl.Execute(rw, data); err != nil {
		log.Fatal(err)
	}
}

func logreq(next http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(rw, r)

		defer log.Printf(
			"%v %v %v %v",
			time.Since(start),
			r.Method,
			r.URL.String(),
			r.Proto,
		)
	}
}

type Link struct {
	ID        string
	URL       string
	CreatedAt *time.Time
}

type LinkService struct {
	DB *sql.DB
}

func (service *LinkService) Links(ctx context.Context, uid string) ([]Link, error) {
	query := `
		SELECT id, url, ul.created_at FROM links AS l
			JOIN user_links AS ul ON l.id = ul.link_id AND ul.user_id = $1
		ORDER BY ul.created_at DESC`

	rows, err := service.DB.QueryContext(ctx, query, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err = rows.Scan(&link.ID, &link.URL, &link.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}
