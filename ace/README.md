# Ace

Ace is a template engine create by [yossi](https://github.com/yosssi/ace), to see the original syntax documentation please [click here](https://github.com/yosssi/ace/blob/master/documentation/syntax.md)

### Basic Example

```pug
// index.ace
= include ./views/partials/header .
h1 {{.Title}}
= include ./views/partials/footer .
```

```pug
// layouts/index.ace
= doctype html
html
  head
    title Main
  body
    {{embed}}
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/ace"
)

func main() {
  // Create a new engine
	engine := html.New("./views", ".ace")

	// Pass the engine to the Views
	app := fiber.New(&fiber.Settings{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) {
    // Render index.ace
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Get("/layout", func(c *fiber.Ctx) {
    // Render index.ace within layouts/main.ace
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	app.Listen(3000)
}

```