# Mustache

Mustache is a template engine created by [hoisie/cbroglie](https://github.com/cbroglie/mustache), to see the original syntax documentation please [click here](https://mustache.github.io/mustache.5.html)

### Basic Example

_**./views/index.mustache**_
```html
{{> views/partials/header }}

<h1>{{Title}}</h1>

{{> views/partials/footer }}
```
_**./views/partials/header.mustache**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.mustache**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.mustache**_
```html
<!DOCTYPE html>
<html>

<head>
  <title>Main</title>
</head>

<body>
  {{{embed}}}
</body>

</html>
```

```go
package main

import (
	"log"
	
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/mustache"
)

func main() {
	// Create a new engine
	engine := mustache.New("./views", ".mustache")

  // Or from an embedded system
  //   Note that with an embedded system the partials included from template files must be
  //   specified relative to the filesystem's root, not the current working directory
  // engine := mustache.NewFileSystem(http.Dir("./views", ".mustache"), ".mustache")

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
