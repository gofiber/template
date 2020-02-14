package amber

import (
	"bytes"
	"html/template"

	"github.com/eknkc/amber"
)

// Render https://github.com/eknkc/amber
func Render(raw string, binding interface{}) (html string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = amber.Compile(raw, amber.DefaultOptions); err != nil {
		return html, err
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return html, err
	}
	html = buf.String()

	return html, err
}
