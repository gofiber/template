module github.com/gofiber/template/mustache/v3

go 1.25.0

require (
	github.com/cbroglie/mustache v1.4.2
	github.com/gofiber/template/v2 v2.1.1
	github.com/gofiber/utils/v2 v2.4.3
	github.com/stretchr/testify v1.12.1
	github.com/valyala/bytebufferpool v1.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/gofiber/template/v2 => ../.
