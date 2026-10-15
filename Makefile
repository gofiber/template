## help: 💡 Display available commands
.PHONY: help
help:
	@echo '⚡️ GoFiber/Template Development:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## audit: 🚀 Conduct quality checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## benchmark: 📈 Benchmark code performance
.PHONY: benchmark
benchmark:
	go test ./... -benchmem -bench=. -run=^Benchmark_$

## coverage: ☂️  Generate coverage report
.PHONY: coverage
coverage:
	go run gotest.tools/gotestsum@v1.12.0 -f testname -- ./... -race -count=1 -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html &

## format: 🎨 Fix code format issues
.PHONY: format
format:
	go run mvdan.cc/gofumpt@latest -w -l .

## benchfmt: 📝 Format README benchmark lines
.PHONY: benchfmt
benchfmt:
	go run ./scripts/format_benchmarks.go

## lint: 🚨 Run lint checks
.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run ./...

## modernize: ♻️ Check for outdated patterns
.PHONY: modernize
modernize:
	go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix -test=false ./...

## test: 🚦 Execute all tests
.PHONY: test
test:
	go run gotest.tools/gotestsum@v1.12.0 -f testname -- ./... -race -count=1 -shuffle=on

## tidy: 📌 Clean and tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy -v
