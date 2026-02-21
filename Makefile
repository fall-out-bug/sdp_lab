# sdp_dev Makefile
.PHONY: test test-internal coverage lint quality

test:
	go test ./... -count=1

test-internal:
	go test ./internal/... -count=1

coverage:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -5

# Coverage for internal only (excludes cmd) - used for 80% gate
coverage-internal:
	@go test ./internal/... -coverprofile=coverage_internal.out -covermode=atomic 2>/dev/null
	@go tool cover -func=coverage_internal.out | grep total

lint:
	golangci-lint run ./...
	go vet ./...

quality: test lint
	@echo "Running sdp quality..."
	@sdp quality all 2>/dev/null || true
