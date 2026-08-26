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

### Partials

A `{{> name }}` include is served from the templates the engine loaded, so a
partial is named exactly the way `Render` names a template: `{{> partials/header }}`
for `./views/partials/header.mustache` under `mustache.New("./views", ".mustache")`.
The directory-qualified form (`{{> views/partials/header }}`) and the root-anchored
form (`{{> /partials/header }}`) resolve to the same file, so templates written
against earlier versions keep working.

With an embedded filesystem the names are relative to the filesystem's root, not
to the working directory, so `//go:embed views` makes the partial above
`{{> views/partials/header }}`. Embedding the contents of `views` instead, or
passing `http.Dir("./views")`, makes it `{{> partials/header }}`.
`mustache.NewFileSystemPartials(fs, ".mustache", partialsFS)` takes the partials
from `partialsFS` as well, for partials that do not sit beside the templates.

Because the partials come from what was loaded, an include cannot reach outside
the engine directory. The leading slash in `{{> /partials/header }}` anchors the
name to that root, not to the filesystem, so a filesystem path such as
`/etc/passwd`, a name that climbs out with `..`, and one that points through a
symbolic link the loader did not walk are all simply not there.
`Load` reports it rather than rendering nothing, naming the template and the
partial, and it reports an include cycle the same way instead of letting the
render recurse. Both are load-time failures, so a bad include shows up when the
engine starts rather than once per request.
