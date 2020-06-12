# Template

![Release](https://img.shields.io/github/release/gofiber/template.svg)
[![Discord](https://img.shields.io/badge/discord-join%20channel-7289DA)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Test/badge.svg)
![Security](https://github.com/gofiber/template/workflows/Security/badge.svg)
![Linter](https://github.com/gofiber/template/workflows/Linter/badge.svg)

This package contains 8 template engines that can be used with [Fiber v1.11.0](https://github.com/gofiber/fiber)
Go version `1.13` or higher is required.

### Installation
```
go get github.com/gofiber/fiber@v1.11.0
go get github.com/gofiber/template
```

### Example usage
```go
package main

import (
	"github.com/gofiber/fiber"

	// "github.com/gofiber/template/ace"
	// "github.com/gofiber/template/amber"
	// "github.com/gofiber/template/django"
	// "github.com/gofiber/template/handlebars"
	// "github.com/gofiber/template/jet"
	// "github.com/gofiber/template/mustache"
	// "github.com/gofiber/template/pug"
	"github.com/gofiber/template/html"
)

func main() {
	// engine := ace.New("./views", ".ace")
	// engine := amber.New("./views", ".amber")
	// engine := django.New("./views", ".django")
	// engine := handlebars.New("./views", ".hbs")
	// engine := jet.New("./views", ".jet")
	// engine := mustache.New("./views", ".mustache")
	// engine := pug.New("./views", ".pug")
  
	engine := html.New("./views", ".html")
	engine.Reload(true)       // reload templates on each render
  engine.Debug(true)        // show parsed templates
  engine.Layout("embed")    // variable name to embed templates, default embed
	engine.Delims("{{", "}}") // custom delimiters
	engine.AddFunc("greet", func(name string) string {
		return "Hello, " + name + "!"
	}) // Add function to global FuncMap

	app := fiber.New(&fiber.Settings{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) {
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main") // Optional layout support using 'embed` variable
	})

	app.Listen(3000)
}

```
