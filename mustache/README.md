---
id: mustache
title: Mustache
---

![Release](https://img.shields.io/github/v/tag/gofiber/template?filter=mustache*)
[![Discord](https://img.shields.io/discord/704680098577514527?style=flat&label=%F0%9F%92%AC%20discord&color=00ACD7)](https://gofiber.io/discord)
![Test](https://github.com/gofiber/template/workflows/Tests%20Mustache/badge.svg)

Mustache is a template engine created by [hoisie/cbroglie](https://github.com/cbroglie/mustache), to see the original syntax documentation please [click here](https://mustache.github.io/mustache.5.html)

## Installation

Go version support: We only support the latest two versions of Go. Visit https://go.dev/doc/devel/release for more information.

```
go get github.com/gofiber/template/mustache/v3
```

## Basic Example

_**./views/index.mustache**_
```html
{{> partials/header }}

<h1>{{Title}}</h1>

{{> partials/footer }}
```
_**./views/partials/header.mustache**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.mustache**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.mustache**_
```html
<!DOCTYPE html>
<html>

<head>
  <title>Main</title>
</head>

<body>
  {{{embed}}}
</body>

</html>
```

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/template/mustache/v3"

	// "net/http" // embedded system
)

func main() {
	// Create a new engine
	engine := mustache.New("./views", ".mustache")

	// Or from an embedded system
	// engine := mustache.NewFileSystem(http.Dir("./views"), ".mustache")

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

## Partials

A `{{> name }}` include is looked up in the engine directory first, so
`{{> partials/header }}` reads `./views/partials/header.mustache` for an engine
created with `mustache.New("./views", ".mustache")`. The name is then tried as
written, which keeps templates that already spell out the full path
(`{{> views/partials/header }}`) working, but only while it still lands inside
the engine directory. With `mustache.NewFileSystem` the lookup is relative to
the filesystem root, which the filesystem confines on its own.

No partial can leave that root: a name that is absolute, that climbs out with
`..`, or that would resolve outside the engine directory is never read. When a
partial is nowhere to be found, rendering fails with an error naming it and
every path that was tried, instead of silently rendering nothing. Set
`engine.Debug(true)` to log each attempt.
