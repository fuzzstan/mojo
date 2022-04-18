package handlers

import (
    "github.com/mojo-lang/mojo/go/pkg/mojo/util"
    "github.com/mojo-lang/mojo/go/pkg/ncraft/data"
    "github.com/mojo-lang/mojo/go/pkg/ncraft/gokit/generator/handlers/templates"
    "github.com/pkg/errors"
    "io"
)

// MiddlewaresPath is the path to the middleware gotemplate file.
const MiddlewaresPath = "pkg/NAME-service/handlers/middlewares.go.tmpl"

// NewMiddlewares returns a Renderable that renders the middlewares.go file.
func NewMiddlewares() *Middlewares {
    var m Middlewares

    return &m
}

// Middlewares satisfies the gengokit.Renderable interface to render
// middlewares.
type Middlewares struct {
    prev io.Reader
}

// Load loads the previous version of the middleware file.
func (m *Middlewares) Load(prev io.Reader) {
    m.prev = prev
}

// Render creates the middlewares.go file. With no previous version it renders
// the templates, if there was a previous version loaded in, it passes that
// through.
func (m *Middlewares) Render(path string, service *data.Service) (io.Reader, error) {
    if path != MiddlewaresPath {
        return nil, errors.Errorf("cannot render unknown file: %q", path)
    }
    if m.prev != nil {
        return m.prev, nil
    }
    return util.ApplyTemplate("Middlewares", templates.Middlewares, service, service.FuncMap)
}
