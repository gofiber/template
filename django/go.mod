module github.com/gofiber/template/django/v4

go 1.25.0

require (
	github.com/flosch/pongo2/v6 v6.1.0
	github.com/gofiber/template/v2 v2.1.0
	github.com/stretchr/testify v1.12.0
)

require (
	github.com/gofiber/utils/v2 v2.4.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gofiber/template/v2 => ../.
