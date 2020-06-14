# HTML

HTML is the official Go template engine [html/template](https://golang.org/pkg/html/template/), to see the original syntax documentation please [click here](https://curtisvermeeren.github.io/2017/09/14/Golang-Templates-Cheatsheet#actions)

### Basic Example

_**./views/index.html**_
```html
{{template "partials/header" .}}

<h1>{{.Title}}</h1>

{{template "partials/footer" .}}
```
_**./views/partials/header.html**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.html**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.html**_
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
	"github.com/gofiber/fiber"
	"github.com/gofiber/template/html"
)

func main() {
	// Create a new engine
	engine := html.New("./views", ".html")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".html"))

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