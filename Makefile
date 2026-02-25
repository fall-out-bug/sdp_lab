# sdp_dev Makefile
# Build outputs: binaries → bin/, coverage → .tmp/
.PHONY: test test-internal test-scripts coverage lint quality generate protocol-e2e
.PHONY: build-sdp-orchestrate build-sdp-guard build-sdp-eval build-sdp-ci-loop build-sdp-evidence build-sdp-ws-verdict-validate
.PHONY: clean-root

test:
	go test ./... -count=1

test-scripts:
	@./scripts/feature_to_pr_test.sh
	@./scripts/oneshot-stop-gate_test.sh

test-internal:
	go test ./internal/... -count=1

# Coverage writes to .tmp/ to keep root clean
coverage:
	@mkdir -p .tmp
	go test ./internal/... -coverprofile=.tmp/coverage.out -covermode=atomic
	go tool cover -func=.tmp/coverage.out | tail -5

# Coverage for internal only (excludes cmd) - used for 80% gate
coverage-internal:
	@mkdir -p .tmp
	@go test ./internal/... -coverprofile=.tmp/coverage_internal.out -covermode=atomic 2>/dev/null
	@go tool cover -func=.tmp/coverage_internal.out | grep total

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

# Phase 0 CLI builds (for local dev)
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

build-sdp-ws-verdict-validate:
	go build -o bin/sdp-ws-verdict-validate ./cmd/sdp-ws-verdict-validate

validate-ws-verdicts:
	@go run ./cmd/sdp-ws-verdict-validate .

# Remove orphaned binaries and coverage files from root (gitignored artifacts)
clean-root:
	@rm -f adapter-controller autonomy-worker beads-fsm brain-gateway cicd-agent
	@rm -f evaluator-orchestrator feature-orchestrator intake-gateway orchestrator
	@rm -f pr-publish registry-agent swarm-orchestrator swarm-worker telemetry-analyzer
	@rm -f sdp-ci-loop sdp-eval sdp-evidence sdp-guard sdp-orchestrate sdp-ws-verdict-validate
	@rm -f *.out coverage*.out adapter_cover.out aw2.out aw_cov.out full_coverage.out swarm_cov.out swarm_coverage.out swarm_qa_coverage.out 2>/dev/null || true
	@echo "Root cleaned."

# Protocol E2E test (Docker)
protocol-e2e:
	@./ci/run-protocol-e2e.sh

# Download envtest binaries for adapter-controller tests (optional)
envtest:
	@go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	@$(shell go env GOPATH)/bin/setup-envtest use -i -p path
