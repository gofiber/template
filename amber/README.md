# Amber

Amber is a template engine create by [eknkc](https://github.com/eknkc/amber), to see the original syntax documentation please [click here](https://github.com/eknkc/amber#tags)

### Basic Example

_**./views/index.amber**_
```html
import ./views/partials/header

h1 #{Title}

import ./views/partials/footer
```
_**./views/partials/header.amber**_
```html
h1 Header
```
_**./views/partials/footer.amber**_
```html
h1 Footer
```
_**./views/layouts/main.amber**_
```html
doctype html
html
  head
    title Main
  body
    #{embed()}
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/amber"
)

func main() {
	// Create a new engine
	engine := html.New("./views", ".amber")

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