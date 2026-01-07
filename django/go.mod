module github.com/gofiber/template/django/v4

go 1.25

require (
	github.com/flosch/pongo2/v6 v6.0.0
	github.com/gofiber/template/v2 v2.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gofiber/utils/v2 v2.0.0-rc.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/gofiber/template/v2 => ../.
