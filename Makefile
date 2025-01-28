.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| cut -d ':' -f 1,2 \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build:
	go build -o ./analyze cmd/analyze/analyze.go

.PHONY: govulncheck
govulncheck: ## Check dependencies for vulnerabilities.
	@command -v govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

.PHONY: lint
lint: lint-gofmt lint-gomod lint-govet ## Run all linters.

.PHONY: lint-gofmt
lint-gofmt: ## Run linter `go fmt`.
ifneq ($(shell gofmt -l . | wc -l),0)
	gofmt -l -d .
	@false
endif

.PHONY: lint-gomod
lint-gomod: ## Run linter `go mod`.
ifneq ($(shell go mod tidy -v 2>/dev/stdout | tee /dev/stderr | grep -c 'unused '),0)
	@false
endif

.PHONY: lint-govet
lint-govet: ## Run linter `go vet`.
	go vet ./...

.PHONY: test
test: ## Run unit tests.
	go clean -testcache
	GIN_MODE="test" go test $(FLAGS) ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and generate code coverage.
	go test $(FLAGS) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
