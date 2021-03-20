# Django

Django is a template engine create by [flosch](https://github.com/flosch/pongo2), to see the original syntax documentation please [click here](https://docs.djangoproject.com/en/dev/topics/templates/)

### Basic Example

_**./views/index.django**_
```html
{% include "partials/header.django" %}

<h1>{{ Title }}</h1>

{% include "partials/footer.django" %}
```
_**./views/partials/header.django**_
```html
<h2>Header</h2>
```
_**./views/partials/footer.django**_
```html
<h2>Footer</h2>
```
_**./views/layouts/main.django**_
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
	"github.com/gofiber/template/django"
)

func main() {
	// Create a new engine
	engine := django.New("./views", ".django")

  // Or from an embedded system
  // See github.com/gofiber/embed for examples
  // engine := html.NewFileSystem(http.Dir("./views", ".django"))

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

### Register and use custom functions
```go
// My custom function
func Nl2brHtml(value interface{}) string {
	if str, ok := value.(string); ok {
		return strings.Replace(str, "\n", "<br />", -1)
	}
	return ""
}

// Create a new engine
engine := django.New("./views", ".django")

// register functions
engine.AddFunc("nl2br", Nl2brHtml)

// Pass the engine to the Views
app := fiber.New(fiber.Config{Views: engine})
```
_**in the handler**_
```go
c.Render("index", fiber.Map{
    "Fiber": "Hello, World!\n\nGreetings from Fiber Team",
})
```

_**./views/index.django**_
```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"></head>
<body>
{{ nl2br(Fiber) }}
</body>
</html>
```
**Output:**
```html
<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"></head>
<body>
Hello, World!<br /><br />Greetings from Fiber Team
</body>
</html>
```
