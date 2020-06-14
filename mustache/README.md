# Mustache

Mustache is a template engine create by [hoisie/cbroglie](https://github.com/cbroglie/mustache), to see the original syntax documentation please [click here](https://mustache.github.io/mustache.5.html)

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
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/mustache"
)

func main() {
	// Create a new engine
	engine := mustache.New("./views", ".mustache")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".mustache"))

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
