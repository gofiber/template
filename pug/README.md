# Pug

Pug is a template engine create by [joker](github.com/Joker/jade), to see the original syntax documentation please [click here](https://pugjs.org/language/tags.html)

### Basic Example

_**./views/index.pug**_
```html
include views/partials/header.pug

h1 #{.Title}

include views/partials/footer.pug
```
_**./views/partials/header.pug**_
```html
h2 Header
```
_**./views/partials/footer.pug**_
```html
h2 Footer
```
_**./views/layouts/main.pug**_
```html
doctype html
html
  head
    title Main
  body
    | {{embed}}
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/pug"
)

func main() {
	// Create a new engine
	engine := pug.New("./views", ".pug")

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
