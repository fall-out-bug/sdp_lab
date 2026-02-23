# sdp_dev Makefile
.PHONY: test test-internal test-scripts coverage lint quality generate

test:
	go test ./... -count=1

test-scripts:
	@./scripts/feature_to_pr_test.sh
	@./scripts/oneshot-stop-gate_test.sh

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

quality: test test-scripts lint
	@echo "Running sdp quality..."
	@if [ -d sdp/sdp-plugin ]; then \
		(cd sdp/sdp-plugin && go build -o /tmp/sdp-quality ./cmd/sdp) && /tmp/sdp-quality quality all; \
	else \
		sdp quality all 2>/dev/null || true; \
	fi

generate:
	$(shell go env GOPATH)/bin/controller-gen object paths=./api/...
	$(shell go env GOPATH)/bin/controller-gen crd paths=./api/... output:crd:dir=deploy/k8s/crd/

# Sync skills from canonical source (sdp/prompts/skills) to copies.
# Use when .opencode/skills and .cursor/skills are real dirs, not symlinks.
sync-skills:
	rsync -a sdp/prompts/skills/ .opencode/skills/
	rsync -a sdp/prompts/skills/ .cursor/skills/ 2>/dev/null || true

# Download envtest binaries for adapter-controller tests (optional)
envtest:
	@go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@$(shell go env GOPATH)/bin/setup-envtest use -i -p path
