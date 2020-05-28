# Template

![Release](https://img.shields.io/github/release/gofiber/template.svg)
[![Discord](https://img.shields.io/badge/discord-join%20channel-7289DA)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Test/badge.svg)
![Security](https://github.com/gofiber/template/workflows/Security/badge.svg)
![Linter](https://github.com/gofiber/template/workflows/Linter/badge.svg)

This package contains 8 template engines that can be used with [Fiber v1.10.0](https://github.com/gofiber/fiber)
Go version `1.13` or higher is required.

### Installation
```
go get github.com/gofiber/fiber@v1.10.0
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
	app := fiber.New()

	// app.Settings.Templates = ace.New("./views", ".ace")
	// app.Settings.Templates = amber.New("./views", ".amber")
	// app.Settings.Templates = django.New("./views", ".django")
	// app.Settings.Templates = handlebars.New("./views", ".hbs")
  	// app.Settings.Templates = jet.New("./views", ".jet")
	// app.Settings.Templates = mustache.New("./views", ".mustache")
	// app.Settings.Templates = pug.New("./views", ".pug")
	app.Settings.Templates = html.New("./views", ".html")

	app.Get("/", func(c *fiber.Ctx) {
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Listen(3000)
}

```
