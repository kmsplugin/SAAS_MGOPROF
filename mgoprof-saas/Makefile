.PHONY: build test test-unit test-integration vet lint

# ── Build ─────────────────────────────────────────────────────────────────────
build:
	go build ./...

# ── Tests ─────────────────────────────────────────────────────────────────────

## Unit tests only (no external DB required)
test-unit:
	go test -race -count=1 ./...

## Integration tests — requires TEST_DATABASE_URL to be set.
## Example:
##   TEST_DATABASE_URL="postgres://user:pass@localhost:5432/mgoprof_test?sslmode=disable" \
##     make test-integration
test-integration:
	@if [ -z "$$TEST_DATABASE_URL" ]; then \
		echo "ERROR: TEST_DATABASE_URL is not set"; \
		echo ""; \
		echo "Usage:"; \
		echo "  TEST_DATABASE_URL=postgres://user:pass@localhost:5432/db?sslmode=disable make test-integration"; \
		exit 1; \
	fi
	go test -race -count=1 -v ./internal/integration/...

## Run both unit + integration (integration skips if TEST_DATABASE_URL unset)
test: test-unit

vet:
	go vet ./...

## Requires golangci-lint to be installed: https://golangci-lint.run/usage/install/
lint:
	golangci-lint run ./...
