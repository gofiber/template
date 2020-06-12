<p align="center">
  <h1>Template</h1>
  <a href="https://github.com/gofiber/fiber/releases">
    <img src="https://img.shields.io/github/v/release/gofiber/template?color=00ACD7&label=%F0%9F%9A%80%20">
  </a>
  <a href="https://pkg.go.dev/github.com/gofiber/template?tab=doc">
    <img src="https://img.shields.io/badge/%F0%9F%93%9A%20godoc-pkg-00ACD7.svg?color=00ACD7&style=flat">
  </a>
  <a href="https://github.com/gofiber/fiber/actions?query=workflow%3AGosec">
    <img src="https://img.shields.io/github/workflow/status/gofiber/template/Security?label=%F0%9F%94%91%20gosec&style=flat&color=75C46B">
  </a>
  <a href="https://github.com/gofiber/fiber/actions?query=workflow%3ATest">
    <img src="https://img.shields.io/github/workflow/status/gofiber/template/Test?label=%F0%9F%A7%AA%20tests&style=flat&color=75C46B">
  </a>
  <a href="https://gofiber.io/discord">
    <img src="https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7">
  </a>
</p>

This package provides universal methods to use multiple template engines with the [Fiber web framework](https://github.com/gofiber/fiber).

8 template engines are supported:
- [html](https://github.com/gofiber/template/tree/master/html)
- [ace](https://github.com/gofiber/template/tree/master/ace)
- [amber](https://github.com/gofiber/template/tree/master/amber)
- [django](https://github.com/gofiber/template/tree/master/django)
- [handlebars](https://github.com/gofiber/template/tree/master/handlebars)
- [jet](https://github.com/gofiber/template/tree/master/jet)
- [mustache](https://github.com/gofiber/template/tree/master/mustache)
- [pug](https://github.com/gofiber/template/tree/master/pug)

### Installation
> Go version `1.13` or higher is required.

```
go get -u github.com/gofiber/fiber
go get -u github.com/gofiber/template
```

### Example
```go
package main

import (
	"github.com/gofiber/fiber"
	// To use a template engine, import as shown below:
	// "github.com/gofiber/template/pug"
	// "github.com/gofiber/template/mustache"

	// In this example we use the html template engine
	"github.com/gofiber/template/html"
)

func main() {
	// Create a new engine by passing the template folder
	// and template extension using <engine>.New(dir, ext string)
	engine := html.New("./views", ".html")

	// Reload the templates on each render, good for development
	engine.Reload(true) // Optional. Default: false

	// Debug will print each template that is parsed, good for debugging
	engine.Debug(true) // Optional. Default: false

	// Layout defines the variable name that is used to yield templates within layouts
	engine.Layout("embed") // Optional. Default: "embed"

	// Delims sets the action delimiters to the specified strings
	engine.Delims("{{", "}}") // Optional. Default: engine delimiters

	// AddFunc adds a function to the template's function map.
	engine.AddFunc("greet", func(name string) string {
		return "Hello, " + name + "!"
	})

	// After you created your engine, you can pass it to Fiber's Views Engine
	app := fiber.New(&fiber.Settings{
		Views: engine,
	})

	// To render a template, you can call the ctx.Render function
	// Render(tmpl string, values interface{}, layout ...string)
	app.Get("/", func(c *fiber.Ctx) {
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	// Render with layout example
	app.Get("/layout", func(c *fiber.Ctx) {
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	app.Listen(3000)
}

```
