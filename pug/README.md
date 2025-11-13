---
id: pug
title: Pug
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=pug*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Pug/badge.svg)

Pug is a template engine create by [joker](https://github.com/Joker/jade), to see the original syntax documentation please [click here](https://pugjs.org/language/tags.html)

## Installation

Go version support: We only support the latest two versions of Go. Visit https://go.dev/doc/devel/release for more information.

```
go get github.com/gofiber/template/pug/v3
```

## Basic Example

_**./views/index.pug**_
```html
include partials/header.pug

h1 #{.Title}

include partials/footer.pug
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
    include ../partials/meta.pug
  body
    | {{embed}}
```

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/pug/v3"

	// "net/http" // embedded system
)

func main() {
	// Create a new engine
	engine := pug.New("./views", ".pug")

	// Or from an embedded system
	// See github.com/gofiber/embed for examples
	// engine := pug.NewFileSystem(http.Dir("./views"), ".pug")

	// Pass the engine to the views
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c fiber.Ctx) error {
		// Render index
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		})
	})

	app.Get("/layout", func(c fiber.Ctx) error {
		// Render index within layouts/main
		return c.Render("index", fiber.Map{
			"Title": "Hello, World!",
		}, "layouts/main")
	})

	log.Fatal(app.Listen(":3000"))
}

```
