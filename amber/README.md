---
id: amber
title: Amber
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=amber*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Amber/badge.svg)

Amber is a template engine create by [eknkc](https://github.com/eknkc/amber), to see the original syntax documentation please [click here](https://github.com/eknkc/amber#tags)

## Installation

Go version support: We only support the latest two versions of Go. Visit https://go.dev/doc/devel/release for more information.

```
go get github.com/gofiber/template/amber/v3
```

## Basic Example

_**./views/index.amber**_
```html
import ./views/partials/header

h1 #{Title}

import ./views/partials/footer
```
_**./views/partials/header.amber**_
```html
h1 Header
```
_**./views/partials/footer.amber**_
```html
h1 Footer
```
_**./views/layouts/main.amber**_
```html
doctype html
html
  head
    title Main
  body
    #{embed()}
```

```go
package main

import (
	"log"
	
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/amber/v3"
)

func main() {
	// Create a new engine
	engine := amber.New("./views", ".amber")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".amber"))

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
