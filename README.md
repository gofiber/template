<p align="center">
  <a href="https://gofiber.io">
    <img alt="Fiber" height="125" src="https://raw.githubusercontent.com/gofiber/docs/master/static/fiber_template_v2_logo.svg">
  </a>
  <br>
  <a href="https://github.com/gofiber/fiber/releases">
    <img src="https://img.shields.io/github/v/release/gofiber/template?color=00ACD7&label=%F0%9F%9A%80%20">
  </a>
  <a href="https://pkg.go.dev/github.com/gofiber/template/html?tab=doc">
    <img src="https://img.shields.io/badge/%F0%9F%93%9A%20godoc-pkg-00ACD7.svg?color=00ACD7&style=flat">
  </a>
  <a href="https://github.com/gofiber/fiber/actions?query=workflow%3ASecurity">
    <img src="https://img.shields.io/github/workflow/status/gofiber/template/Security?label=%F0%9F%94%91%20gosec&style=flat&color=75C46B">
  </a>
  <a href="https://github.com/gofiber/fiber/actions?query=workflow%3ATest">
    <img src="https://img.shields.io/github/workflow/status/gofiber/template/Test?label=%F0%9F%A7%AA%20tests&style=flat&color=75C46B">
  </a>
  <a href="https://gofiber.io/discord">
    <img src="https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7">
  </a>
</p>

This package provides universal methods to use multiple template engines with the [Fiber web framework](https://github.com/gofiber/fiber) using the new [Views](https://godoc.org/github.com/gofiber/fiber#Views) interface that is available from `> v1.11.1`. Special thanks to @bdtomlin & @arsmn for helping!

8 template engines are supported:
- [html](https://github.com/gofiber/template/tree/master/html)
- [ace](https://github.com/gofiber/template/tree/master/ace)
- [amber](https://github.com/gofiber/template/tree/master/amber)
- [django](https://github.com/gofiber/template/tree/master/django)
- [handlebars](https://github.com/gofiber/template/tree/master/handlebars)
- [jet](https://github.com/gofiber/template/tree/master/jet)
- [mustache](https://github.com/gofiber/template/tree/master/mustache)
- [pug](https://github.com/gofiber/template/tree/master/pug)

### Installation
> Go version `1.16` or higher is required.

```
go get -u github.com/gofiber/fiber/v2
go get -u github.com/gofiber/template
```

### Example
```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	// To use a specific template engine, import as shown below:
	// "github.com/gofiber/template/pug"
	// "github.com/gofiber/template/mustache"
	// etc..

	// In this example we use the html template engine
	"github.com/gofiber/template/html"
)

func main() {
	// Create a new engine by passing the template folder
	// and template extension using <engine>.New(dir, ext string)
	engine := html.New("./views", ".html")

  	// We also support the http.FileSystem interface
	// See examples below to load templates from embedded files
	engine := html.NewFileSystem(http.Dir("./views"), ".html")

	// Reload the templates on each render, good for development
	engine.Reload(true) // Optional. Default: false

	// Debug will print each template that is parsed, good for debugging
	engine.Debug(true) // Optional. Default: false

	// Layout defines the variable name that is used to yield templates within layouts
	engine.Layout("embed") // Optional. Default: "embed"

	// Delims sets the action delimiters to the specified strings
	engine.Delims("{{", "}}") // Optional. Default: engine delimiters

	// AddFunc adds a function to the template's global function map.
	engine.AddFunc("greet", func(name string) string {
		return "Hello, " + name + "!"
	})

	// After you created your engine, you can pass it to Fiber's Views Engine
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// To render a template, you can call the ctx.Render function
	// Render(tmpl string, values interface{}, layout ...string)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	// Render with layout example
	app.Get("/layout", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	log.Fatal(app.Listen(":3000"))
}

```

### More Examples

To view more specific examples, you could visit each engine folder to learn more
- [html](https://github.com/gofiber/template/tree/master/html)
- [ace](https://github.com/gofiber/template/tree/master/ace)
- [amber](https://github.com/gofiber/template/tree/master/amber)
- [django](https://github.com/gofiber/template/tree/master/django)
- [handlebars](https://github.com/gofiber/template/tree/master/handlebars)
- [jet](https://github.com/gofiber/template/tree/master/jet)
- [mustache](https://github.com/gofiber/template/tree/master/mustache)
- [pug](https://github.com/gofiber/template/tree/master/pug)


### embedded Systems

We support the `http.FileSystem` interface, so you can use different libraries to load the templates from embedded binaries.

#### pkger
Read documentation: https://github.com/markbates/pkger

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"

	"github.com/markbates/pkger"
)

func main() {
	engine := html.NewFileSystem(pkger.Dir("/views"), ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// run pkger && go build
}
```
#### packr
Read documentation: https://github.com/gobuffalo/packr

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"

	"github.com/gobuffalo/packr/v2"
)

func main() {
	engine := html.NewFileSystem(packr.New("Templates", "/views"), ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// run packr && go build
}
```
#### go.rice
Read documentation: https://github.com/GeertJohan/go.rice

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"

	"github.com/GeertJohan/go.rice"
)

func main() {
	engine := html.NewFileSystem(rice.MustFindBox("views").HTTPBox(), ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// run rice embed-go && go build
}

```
#### fileb0x
Read documentation: https://github.com/UnnoTed/fileb0x

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	// your generated package
	"github.com/<user>/<repo>/static"
)

func main() {
	engine := html.NewFileSystem(static.HTTP, ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Read the documentation on how to use fileb0x
}
```
