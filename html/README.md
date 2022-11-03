# HTML

HTML is the official Go template engine [html/template](https://golang.org/pkg/html/template/), to see the original syntax documentation please [click here](TEMPLATES_CHEATCHEET.md)

**Info:**

All templates within the specified view directory are analyzed and compiled at the beginning to increase the performance when using them.
Thus it should be noted that no `definition` with the same name should exist, otherwise they will overwrite each other.
For templating the `{{embed}}` tag should be used

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
	"log"
	
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
)

func main() {
	// Create a new engine
	engine := html.New("./views", ".html")

	// Or from an embedded system
	// See github.com/gofiber/embed for examples
	// engine := html.NewFileSystem(http.Dir("./views", ".html"))

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

### Example with embed.FS

```go
package main

import (
    "log"
    "net/http"
    "embed"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/template/html"
)

//go:embed views/*
var viewsfs embed.FS

func main() {
    engine := html.NewFileSystem(http.FS(viewsfs), ".html")

    // Pass the engine to the Views
    app := fiber.New(fiber.Config{
        Views: engine,
    })


    app.Get("/", func(c *fiber.Ctx) error {
        // Render index - start with views directory
        return c.Render("views/index", fiber.Map{
            "Title": "Hello, World!",
        })
    })

    log.Fatal(app.Listen(":3000"))
}
```

and change the starting point to the views directory

_**./views/index.html**_
```html
{{template "views/partials/header" .}}

<h1>{{.Title}}</h1>

{{template "views/partials/footer" .}}
```

### Example with innerHTML

```go
package main

import (
    "embed"
    "html/template"
    "log"
    "net/http"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/template/html"
)

//go:embed views/*
var viewsfs embed.FS

func main() {
    engine := html.NewFileSystem(http.FS(viewsfs), ".html")
    engine.AddFunc(
        // add unescape function
        "unescape", func(s string) template.HTML {
            return template.HTML(s)
        },
    )

    // Pass the engine to the Views
    app := fiber.New(fiber.Config{Views: engine})

    app.Get("/", func(c *fiber.Ctx) error {
        // Render index
        return c.Render("views/index", fiber.Map{
            "Title": "Hello, <b>World</b>!",
        })
    })

    log.Fatal(app.Listen(":3000"))
}
```

and change the starting point to the views directory

_**./views/index.html**_
```html
<p>{{ unescape .Title}}</p>
```
**html output**
```html
<p>Hello, <b>World</b>!</p>
```

