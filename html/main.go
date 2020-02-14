package html

import (
	"bytes"
	"html/template"
)

// Render https://golang.org/pkg/text/template/
func Render(raw string, binding interface{}) (html string, err error) {
	var buf bytes.Buffer
	var tmpl *template.Template

	if tmpl, err = template.New("").Parse(raw); err != nil {
		return html, err
	}
	if err = tmpl.Execute(&buf, binding); err != nil {
		return html, err
	}
	html = buf.String()

	return html, err
}
