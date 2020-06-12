# Jet

Jet is a template engine create by [cloudykit](github.com/CloudyKit/jet), to see the original syntax documentation please [click here](https://github.com/CloudyKit/jet/wiki/3.-Jet-template-syntax)

### Basic Example

_**./views/index.jet**_
```html
{{include "partials/header"}}

<h1>{{ Title }}</h1>

{{include "partials/footer"}}
```
_**./views/partials/header.jet**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.jet**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.jet**_
```html
<!DOCTYPE html>
<html>

<head>
  <title>Title</title>
</head>

<body>
  {{ embed() }}
</body>

</html>
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/jet"
)

func main() {
	// Create a new engine
	engine := html.New("./views", ".jet")

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