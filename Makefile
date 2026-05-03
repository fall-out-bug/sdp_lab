# github.com/fall-out-bug/sdp_lab Makefile
.PHONY: test test-internal test-scripts coverage lint quality quality-go generate protocol-e2e
.PHONY: build-sdp build-sdp-orchestrate build-sdp-guard build-sdp-eval build-sdp-ci-loop build-sdp-evidence
.PHONY: install-hooks f145-demo

GO_TAGS = -tags "sqlite_fts5"

test:
	go test ./... -count=1 $(GO_TAGS)

test-scripts:
	@./scripts/feature_to_pr_test.sh
	@./scripts/oneshot-stop-gate_test.sh

test-internal:
	go test ./internal/... -count=1 $(GO_TAGS)

coverage:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic $(GO_TAGS)
	go tool cover -func=coverage.out | tail -5

# Coverage for internal only (excludes cmd) - maturity-tiered targets:
#   happy-path >= 80%, GA >= 60%, Beta >= 50% (advisory), Experimental exempt
coverage-internal:
	@go test ./internal/... -coverprofile=coverage_internal.out -covermode=atomic $(GO_TAGS) 2>/dev/null
	@go tool cover -func=coverage_internal.out | grep total

lint:
	golangci-lint run ./...
	go vet ./...

quality-go:
	@./scripts/run_go_quality_gates.sh

quality: quality-go test-scripts
	@echo "Running sdp quality..."
	@if [ -d sdp/sdp-plugin ]; then \
		(cd sdp/sdp-plugin && go build -o /tmp/sdp-quality ./cmd/sdp) && /tmp/sdp-quality quality all; \
	else \
		sdp quality all 2>/dev/null || true; \
	fi

generate:
	$(shell go env GOPATH)/bin/controller-gen object paths=./api/...
	$(shell go env GOPATH)/bin/controller-gen crd paths=./api/... output:crd:dir=deploy/k8s/crd/

# Sync skills from canonical source (prompts/skills) to copies.
# Use when .opencode/skills and .cursor/skills are real dirs, not symlinks.
sync-skills:
	rsync -a prompts/skills/ .opencode/skills/
	rsync -a prompts/skills/ .cursor/skills/ 2>/dev/null || true

# Phase 0 CLI builds (for local dev)
build-sdp:
	go build -o bin/sdp $(GO_TAGS) ./cmd/sdp

build-sdp-orchestrate:
	go build -o bin/sdp-orchestrate ./cmd/sdp-orchestrate

build-sdp-guard:
	go build -o bin/sdp-guard ./cmd/sdp-guard

build-sdp-eval:
	go build -o bin/sdp-eval ./cmd/sdp-eval

build-sdp-ci-loop:
	go build -o bin/sdp-ci-loop ./cmd/sdp-ci-loop

build-sdp-evidence:
	go build -o bin/sdp-evidence ./cmd/sdp-evidence

# Protocol E2E test (Docker)
protocol-e2e:
	@./ci/run-protocol-e2e.sh

# Download envtest binaries for adapter-controller tests (optional)
envtest:
	@go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@$(shell go env GOPATH)/bin/setup-envtest use -i -p path

install-hooks:
	bash scripts/install-hooks.sh

# F145 Day-8 acceptance demo: builds cascade-replay and runs on test corpus
# Passes if stayed-cheap >= 40%, fails otherwise.
f145-demo:
	@./scripts/f145-demo.sh

# Sync .pi/ resources into pi-sdp-harness/ package for distribution
sync-pi-package:
	@echo "Syncing pi-sdp-harness package..."
	@mkdir -p pi-sdp-harness/{extensions,skills,prompts,themes}
	@cp -r .pi/extensions/* pi-sdp-harness/extensions/
	@cp -r .pi/skills/* pi-sdp-harness/skills/
	@cp -r .pi/prompts/* pi-sdp-harness/prompts/
	@echo "Synced: extensions, skills, prompts"
	@echo "Version: $$(jq -r .version pi-sdp-harness/package.json)"

# Validate Pi harness consistency with sdp.manifest.yaml
check-pi-harness:
	@./scripts/check-pi-harness.sh
