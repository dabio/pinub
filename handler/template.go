package handler

import (
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sync"
	"text/template"
)

const (
	contentType = "text/html; charset=utf8"
	layoutTpl   = "_layout.html"
)

type tpl struct {
	tplPath string
	init    sync.Once
	html    map[string]*template.Template
}

func newTpl(tplPath string) *tpl {
	t := &tpl{tplPath: tplPath}

	return t
}

func (t *tpl) initTpl() {
	log.Println("initTpl")
	t.html = make(map[string]*template.Template)
	layout := template.Must(template.New(layoutTpl).Funcs(template.FuncMap{}).ParseFiles(t.tplPath + "/" + layoutTpl))

	files, _ := filepath.Glob(t.tplPath + "*.html")
	for _, f := range files {
		if path.Base(f) == layoutTpl {
			continue
		}
		t.html[path.Base(f)] = template.Must(template.Must(layout.Clone()).ParseFiles(f))
	}
}

func (t *tpl) render(w http.ResponseWriter, name string, data interface{}) {
	t.init.Do(func() {
		t.initTpl()
	})

	w.Header().Set("Content-Type", contentType)
	if err := t.html[name].Execute(w, data); err != nil {
		log.Fatal(err)
	}
}
