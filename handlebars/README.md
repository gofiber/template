---
id: handlebars
title: Handlebars
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=handlebars*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Handlebars/badge.svg)

Handlebars is a template engine create by [aymerick](https://github.com/aymerick/raymond), to see the original syntax documentation please [click here](https://github.com/aymerick/raymond#table-of-contents)

### Basic Example

_**./views/index.hbs**_
```html
{{> 'partials/header' }}

<h1>{{Title}}</h1>

{{> 'partials/footer' }}
```
_**./views/partials/header.hbs**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.hbs**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.hbs**_
```html
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
	"log"
	
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/handlebars/v2"
)

func main() {
	// Create a new engine
	engine := handlebars.New("./views", ".hbs")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".hbs"))

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
