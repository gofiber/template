### Install
```
go get -u github.com/gofiber/fiber
go get -u github.com/gofiber/template
```
### Example
*./views/index.mustache*
```
<html>
<head><title>Template Demo</title></head>
<body>
Hi, my name is {{{name}}} and im {{{age}}} years old
</body>
</html>
```

```go
package main

import (
	"github.com/gofiber/fiber"
	"github.com/gofiber/template"
)

func main() {
  app := fiber.New()
  
  app.Settings.TemplateFolder = "./views"
  app.Settings.TemplateExtension = ".mustache"

  // app.Settings.TemplateEngine = template.Amber
  // app.Settings.TemplateEngine = template.Handlebars
  // app.Settings.TemplateEngine = template.Pug
  app.Settings.TemplateEngine = template.Mustache

	app.Get("/", func(c *fiber.Ctx) {
		bind := fiber.Map{
			"name": "John",
			"age":  "35",
		}
		if err := c.Render("index", bind); err != nil {
			c.Status(500).Send(err.Error())
		}
	})

	app.Listen(3000)
}
```
### Test
```curl
curl http://localhost:3000
```
