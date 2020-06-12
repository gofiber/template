# Handlebars

Handlebars is a template engine create by [aymerick](https://github.com/aymerick/raymond), to see the original syntax documentation please [click here](https://github.com/aymerick/raymond#table-of-contents)

### Basic Example

_**./views/index.hbs**_
```pug
{{> 'partials/header' }}

<h1>{{Title}}</h1>

{{> 'partials/footer' }}
```
_**./views/partials/header.hbs**_
```pug
<h2>Header</h2>
```
_**./views/partials/footer.hbs**_
```pug
<h2>Footer</h2>
```
_**./views/layouts/main.hbs**_
```pug
<!DOCTYPE html>
<html>

<head>
  <title>Main</title>
</head>

<body>
  {{embed}}
</body>

</html>
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/handlebars"
)

func main() {
	// Create a new engine
	engine := html.New("./views", ".hbs")

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