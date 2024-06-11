package templating

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

type Templates struct {
	Templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func NewTemplate() *Templates {
	return &Templates{
		Templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

type FormData struct {
	Values map[string]string
	Errors map[string]string
}

func NewFormData() FormData {
	return FormData{
		Values: make(map[string]string),
		Errors: make(map[string]string),
	}
}
