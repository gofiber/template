---
id: jet
title: Jet
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=jet*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Jet/badge.svg)

Jet is a template engine create by [cloudykit](https://github.com/CloudyKit/jet), to see the original syntax documentation please [click here](https://github.com/CloudyKit/jet/wiki/3.-Jet-template-syntax)

## Installation

Go version support: We only support the latest two versions of Go. Visit https://go.dev/doc/devel/release for more information.

```
go get github.com/gofiber/template/jet/v3
```

## Basic Example

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
	"log"
	
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/jet/v3"
)

func main() {
	// Create a new engine
	engine := jet.New("./views", ".jet")

	// Or from an embedded system
	// See github.com/gofiber/embed for examples
	// engine := jet.NewFileSystem(http.Dir("./views"), ".jet")

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
