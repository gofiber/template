# Pug

Pug is a template engine create by [joker](https://github.com/Joker/jade), to see the original syntax documentation please [click here](https://pugjs.org/language/tags.html)

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
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/pug"

	// "net/http" // embedded system
)

func main() {
	// Create a new engine
	engine := pug.New("./views", ".pug")

	// Or from an embedded system
	// See github.com/gofiber/embed for examples
	// engine := pug.NewFileSystem(http.Dir("./views", ".pug"))

	// Pass the engine to the Views
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		// Render index
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Get("/layout", func(c *fiber.Ctx) error {
		// Render index within layouts/main
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	log.Fatal(app.Listen(":3000"))
}

```
