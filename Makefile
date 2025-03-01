.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| cut -d ':' -f 1,2 \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: warm_cache
	go build -o ./what cmd/analyze/main.go

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
lint: lint-gofmt lint-gomod lint-govet lint-staticcheck ## Run all linters.

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

.PHONY: lint-staticcheck
lint-staticcheck: ## Run linter `staticcheck`.
	go tool staticcheck ./...

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

.PHONY: profile-cpu
profile-cpu: ## Collect CPU profile in filename: cpu.pprof
	go test $(FLAGS) -cpuprofile cpu.pprof -count=1 ./pkg/rules

.PHONY: profile-mem
profile-mem: ## Collect memory profile in filename: mem.pprof
	go test $(FLAGS) -memprofile mem.pprof -count=1 ./pkg/rules

.PHONY: profile-mutex
profile-mutex: ## Collect mutex profile in filename: mutex.pprof
	go test $(FLAGS) -memprofile mem.pprof -count=1 ./pkg/rules
