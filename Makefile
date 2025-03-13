GOLANGCI_LINT_VERSION=v1.64

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| cut -d ':' -f 1,2 \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: warm_cache
	go build -o what ./cmd/what

.PHONY: gen_docs
gen_docs: ## Generates CEL function documentation
	go run cmd/gen_docs/main.go docs/functions.md

.PHONY: warm_cache
warm_cache: ## Warms the expression cache (run this when expressions change).
	go run cmd/warm_cache/main.go expr.cache

.PHONY: govulncheck
govulncheck: ## Check dependencies for vulnerabilities.
	go tool govulncheck ./...

.PHONY: lint
lint: lint-gomod lint-golangci ## Run all linters.

.PHONY: lint-gomod
lint-gomod: ## Run linter `go mod`.
ifneq ($(shell go mod tidy -v 2>/dev/stdout | tee /dev/stderr | grep -c 'unused '),0)
	@false
endif

.PHONY: lint-golangci
lint-golangci:
	command -v golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	golangci-lint run

.PHONY: test
test: ## Run unit tests.
	go test $(FLAGS) -race -count=1 ./...

.PHONY: bench-light
bench-light: ## Run a single benchmark on the test filesystem.
	go test $(FLAGS) -run=Analyze -bench=Analyze_TestFS ./...

.PHONY: bench
bench: ## Run benchmarks.
	go test $(FLAGS) -run=Analyze -bench=Analyze -cpu 1,2,4,8 ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and generate code coverage.
	go test $(FLAGS) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: profile
profile: ## Collect profiles saved as *.pprof.
	go test $(FLAGS) -cpuprofile cpu.pprof -bench=Analyze_TestFS ./pkg/rules
	go test $(FLAGS) -memprofile mem.pprof -bench=Analyze_TestFS ./pkg/rules
	go test $(FLAGS) -mutexprofile mutex.pprof -bench=Analyze_TestFS ./pkg/rules

.PHONY: clean
clean: ## Clean files generated from builds and tests.
	rm -f *.pprof what coverage.out rules.test
