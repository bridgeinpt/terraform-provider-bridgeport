default: build

# One-time bootstrap: resolve deps (including the BridgePort Go SDK
# pseudo-version) and write go.sum. Run this once after cloning a fresh
# scaffold, then commit go.mod + go.sum.
.PHONY: bootstrap
bootstrap:
	go mod tidy

.PHONY: build
build:
	go build -v ./...

.PHONY: install
install:
	go install -v ./...

# Static checks. Requires golangci-lint (https://golangci-lint.run/).
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -s -w -e .

.PHONY: vet
vet:
	go vet ./...

# Unit tests (fast, no live instance).
.PHONY: test
test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# Acceptance tests. Spins up a disposable BridgePort, mints a token, runs the
# TF_ACC suite, and tears down. Requires Docker + Terraform on PATH.
.PHONY: testacc
testacc:
	./scripts/acc-harness.sh test

# Regenerate the registry docs under docs/ from the schema + examples/.
# Requires tfplugindocs (fetched on demand via `go run`).
.PHONY: generate
generate:
	go generate ./...

.PHONY: tidy
tidy:
	go mod tidy
