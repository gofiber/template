# Ace

Ace is a template engine create by [yossi](https://github.com/yosssi/ace), to see the original syntax documentation please [click here](https://github.com/yosssi/ace/blob/master/documentation/syntax.md)

### Basic Example

_**./views/index.ace**_
```html
= include ./views/partials/header .

h1 {{.Title}}

= include ./views/partials/footer .
```
_**./views/partials/header.ace**_
```html
h1 Header
```
_**./views/partials/footer.ace**_
```html
h1 Footer
```
_**./views/layouts/main.ace**_
```html
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
	engine := ace.New("./views", ".ace")

	// Pass the engine to the Views
	app := fiber.New(&fiber.Settings{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) {
		// Render index
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Get("/layout", func(c *fiber.Ctx) {
		// Render index within layouts/main
		c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	app.Listen(3000)
}

```
