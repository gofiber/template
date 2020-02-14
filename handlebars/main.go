package handlebars

import "github.com/aymerick/raymond"

// Render https://github.com/aymerick/raymond
func Render(raw string, data interface{}) (html string, err error) {
	return raymond.Render(raw, data)
}
