package mustache

import "github.com/cbroglie/mustache"

// Render https://github.com/hoisie/mustache
func Render(raw string, binding interface{}) (html string, err error) {
	return mustache.Render(raw, binding)
}
