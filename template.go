package template

import (
	"bytes"
	"html/template"

	"github.com/Joker/jade"
	"github.com/aymerick/raymond"
	"github.com/cbroglie/mustache"
	"github.com/eknkc/amber"
)

// Amber https://github.com/eknkc/amber
func Amber(raw string, binding interface{}) (out string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = amber.Compile(raw, amber.DefaultOptions); err != nil {
		return out, err
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return out, err
	}
	out = buf.String()

	return out, err
}

// Handlebars https://github.com/aymerick/raymond
func Handlebars(raw string, data interface{}) (out string, err error) {
	return raymond.Render(raw, data)
}

// HTML https://golang.org/pkg/text/template/
func HTML(raw string, binding interface{}) (out string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = template.New("").Parse(raw); err != nil {
		return out, err
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return out, err
	}
	out = buf.String()

	return out, err
}

// Mustache https://github.com/hoisie/mustache
func Mustache(raw string, binding interface{}) (html string, err error) {
	return mustache.Render(raw, binding)
}

// Pug https://github.com/Joker/jade
func Pug(raw string, binding interface{}) (html string, err error) {
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
