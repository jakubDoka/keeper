package pages

import (
	"embed"
	"sync"

	"github.com/jakubDoka/keeper/util/khtml"
)

//go:embed js/* css/*
var Static embed.FS

var pool = sync.Pool{
	New: func() interface{} {
		return &Page{}
	},
}

type Page struct {
	khtml.Html
	styles, scripts []string
}

func Get() *Page {
	return pool.Get().(*Page)
}

func Put(p *Page) {
	pool.Put(p)
}

func (p *Page) Base(body *Page, buff []byte) []byte {
	p.
		Doctype("html").
		Tag("html", "lang", "en").
		Tag("head").
		Tag("meta", "charset", "utf-8").End().
		Tag("meta", "http-equiv", "X-UA-Compatible", "content", "IE=edge").End().
		Tag("meta", "name", "viewport", "content", "width=device-width, initial-scale=1").End().
		Style("css/main.css")
	for i := len(p.styles) - 1; i >= 0; i-- {
		p.Style(p.styles[i])
	}
	p.End().Tag("body").Wrap(&body.Html)
	Put(body)
	p.Script("js/main.js")
	for i := len(p.scripts) - 1; i >= 0; i-- {
		p.Script(p.scripts[i])
	}

	return p.Close(buff)
}

func (p *Page) AddStyle(s ...string) {
	p.styles = append(p.styles, s...)
}

func (p *Page) AddScript(s ...string) {
	p.scripts = append(p.scripts, s...)
}
