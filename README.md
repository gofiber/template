---
title: 👋 Welcome
description: 🧬 Template engine middlewares for Fiber.
sidebar_position: 1
---


<p align="center">
  <img height="125" alt="Fiber" src="https://raw.githubusercontent.com/gofiber/template/master/.github/logo-dark.svg#gh-dark-mode-only"/>
  <img height="125" alt="Fiber" src="https://raw.githubusercontent.com/gofiber/template/master/.github/logo.svg#gh-light-mode-only" />
  <br/>

  <a href="https://pkg.go.dev/github.com/gofiber/template?tab=doc">
    <img src="https://img.shields.io/badge/%F0%9F%93%9A%20godoc-pkg-00ACD7.svg?color=00ACD7&style=flat"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/gofiber/template">
    <img src="https://img.shields.io/badge/%F0%9F%93%9D%20goreport-A%2B-75C46B"/>
  </a>
  <a href="https://gofiber.io/discord">
    <img src="https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7"/>
  </a>
</p>

This package provides universal methods to use multiple template engines with the [Fiber web framework](https://github.com/gofiber/fiber) using the new [Views](https://godoc.org/github.com/gofiber/fiber#Views) interface that is available from `> v1.11.1`. Special thanks to @bdtomlin & @arsmn for helping!

9 template engines are supported:
- [ace](./ace/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Ace%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-ace.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a> 
- [amber](./amber/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Amber%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-amber.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>
- [django](./django/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Django%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-django.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a> 
- [handlebars](./handlebars/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Handlebars%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-handlebars.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>
- [html](./html/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Html%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-html.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/> </a>
- [jet](./jet/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Jet%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-jet.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>
- [mustache](./mustache/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Mustache%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-mustache.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>
- [pug](./pug/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Pug%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-pug.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>
- [slim](./slim/README.md) <a href="https://github.com/gofiber/template/actions?query=workflow%3A%22Tests+Slim%22"> <img src="https://img.shields.io/github/actions/workflow/status/gofiber/template/test-slim.yml?branch=master&label=%F0%9F%A7%AA%20&style=flat&color=75C46B"/></a>

### Installation
> Go version `1.17` or higher is required.

```
go get -u github.com/gofiber/fiber/v2
go get -u github.com/gofiber/template/any_template_engine/vX
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
	"github.com/gofiber/template/html/v2"
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
- [ace](./ace/README.md)
- [amber](./amber/README.md)
- [django](./django/README.md)
- [handlebars](./handlebars/README.md)
- [html](./html/README.md)
- [jet](./jet/README.md)
- [mustache](./mustache/README.md)
- [pug](./pug/README.md)
- [slim](./slim/README.md)


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


### Benchmarks

#### Simple
![](https://raw.githubusercontent.com/gofiber/template/master/.github/data/Simple-TimeperOperation.png)

#### Extended
![](https://raw.githubusercontent.com/gofiber/template/master/.github/data/Extended-TimeperOperation.png)

Benchmarks were ran on Apple Macbook M1. Each engine was benchmarked 20 times and the results averaged into a single xlsx file. Mustache was excluded from the extended benchmark
