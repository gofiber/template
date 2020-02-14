package pug

import (
	"bytes"
	"html/template"

	"github.com/Joker/jade"
)

// Render https://github.com/Joker/jade
func Render(raw string, binding interface{}) (html string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if raw, err = jade.Parse("", []byte(raw)); err != nil {
		return html, err
	}
	if tmpl, err = template.New("").Parse(raw); err != nil {
		return html, err
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return html, err
	}
	html = buf.String()

	return html, err

}
