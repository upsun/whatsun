GOLANGCI_LINT_VERSION := v1.64
BUILD_FLAGS := -trimpath -ldflags='-s'

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| cut -d ':' -f 1,2 \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: warm_cache ## Build the 'whatsun' binary.
	go build $(BUILD_FLAGS) -o whatsun ./cmd/whatsun

.PHONY: gen_docs
gen_docs: ## Generate CEL function documentation.
	go run cmd/gen_docs/main.go docs/functions.md

.PHONY: warm_cache
warm_cache:
	go run cmd/warm_cache/main.go expr.cache

.PHONY: govulncheck
govulncheck: ## Check dependencies for vulnerabilities.
	go tool govulncheck ./...

.PHONY: lint
lint: lint-gomod lint-golangci ## Run linters.

.PHONY: lint-gomod
lint-gomod:
ifneq ($(shell go mod tidy -v 2>/dev/stdout | tee /dev/stderr | grep -c 'unused '),0)
	@false
endif

.PHONY: lint-golangci
lint-golangci:
	command -v golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	golangci-lint run --timeout=2m

.PHONY: test
test: ## Run unit tests.
	go test -race -count=1 ./...

.PHONY: bench
bench: ## Run benchmarks.
	go test -run=Digest -bench=Digest -cpu 1,2,4,8 ./...

.PHONY: bench-light
bench-light: ## Run a single benchmark on the test filesystem.
	go test -run=Digest -bench=Digest ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and generate a code coverage report.
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: profile
profile: ## Collect profiles saved as *.pprof.
	go test -cpuprofile cpu.pprof -bench=Digest ./pkg/files
	go test -memprofile mem.pprof -bench=Digest ./pkg/files
	go test -mutexprofile mutex.pprof -bench=Digest ./pkg/files

.PHONY: clean
clean: ## Delete files generated from builds and tests.
	rm -f *.pprof whatsun coverage.out rules.test
