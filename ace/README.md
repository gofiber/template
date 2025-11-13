---
id: ace
title: Ace
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=ace*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Ace/badge.svg)

Ace is a template engine create by [yossi](https://github.com/yosssi/ace), to see the original syntax documentation please [click here](https://github.com/yosssi/ace/blob/master/documentation/syntax.md)

## Installation

Go version support: We only support the latest two versions of Go. Visit https://go.dev/doc/devel/release for more information.

```
go get github.com/gofiber/template/ace/v3
```

## Basic Example

_**./views/index.ace**_
```html
= include ./views/partials/header .

h1 {{.Title}}

= include ./views/partials/footer .
```
_**./views/partials/header.ace**_
```html
h1 Header
```
_**./views/partials/footer.ace**_
```html
h1 Footer
```
_**./views/layouts/main.ace**_
```html
= doctype html
html
  head
    title Main
  body
    {{embed}}
```

```go
package main

import (
	"log"
	
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/ace/v3"
)

func main() {
	// Create a new engine
	engine := ace.New("./views", ".ace")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".ace"))

	// Pass the engine to the Views
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
