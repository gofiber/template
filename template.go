package template

import (
	"bytes"
	"html/template"

	pug "github.com/Joker/jade"
	handlebars "github.com/aymerick/raymond"
	mustache "github.com/cbroglie/mustache"
	amber "github.com/eknkc/amber"
)

// Amber https://github.com/eknkc/amber
func Amber(raw string, binding interface{}) (out string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = amber.Compile(raw, amber.DefaultOptions); err != nil {
		return
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return
	}
	out = buf.String()

	return
}

// Handlebars https://github.com/aymerick/raymond
func Handlebars(raw string, data interface{}) (out string, err error) {
	return handlebars.Render(raw, data)
}

// HTML https://golang.org/pkg/text/template/
func HTML(raw string, binding interface{}) (out string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = template.New("").Parse(raw); err != nil {
		return
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return
	}
	out = buf.String()
	return
}

// Mustache https://github.com/hoisie/mustache
func Mustache(raw string, binding interface{}) (out string, err error) {
	return mustache.Render(raw, binding)
}

// Pug https://github.com/Joker/jade
func Pug(raw string, binding interface{}) (out string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if raw, err = pug.Parse("", []byte(raw)); err != nil {
		return
	}
	if tmpl, err = template.New("").Parse(raw); err != nil {
		return
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return
	}
	out = buf.String()
	return

}
