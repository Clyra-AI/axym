SHELL := /bin/bash

.PHONY: lint-fast test-fast test-contracts test-scenarios test-hardening test-chaos test-perf test-adapter-parity test-docs-consistency test-docs-storyline prepush codeql release-local prepush-full

lint-fast:
	@test -z "$$(gofmt -l .)" || (echo "gofmt required"; gofmt -l .; exit 1)
	@go vet ./...

test-fast:
	@go test ./... -count=1

test-contracts:
	@go test ./testinfra/contracts/... -count=1

test-scenarios:
	@go test ./scenarios/... -count=1

test-hardening:
	@go test ./... -count=1 -run TestHardening

test-chaos:
	@go test ./... -count=1 -run TestChaos

test-perf:
	@go test ./... -count=1 -run TestPerf

test-adapter-parity:
	@go test ./... -count=1 -run TestAdapterParity

test-docs-consistency:
	@./scripts/check_docs_consistency.sh

test-docs-storyline:
	@./scripts/check_docs_storyline.sh

prepush: lint-fast test-fast test-contracts

codeql:
	@command -v codeql >/dev/null || (echo "codeql CLI is required"; exit 7)
	@rm -rf .codeql
	@mkdir -p .codeql
	@codeql database create .codeql/db --language=go --source-root=. --command='go build ./cmd/axym'
	@codeql database analyze .codeql/db codeql/go-queries:codeql-suites/go-security-and-quality.qls --format=sarif-latest --output .codeql/results.sarif

release-local:
	@mkdir -p dist
	@go build -o dist/axym ./cmd/axym
	@(cd dist && PATH="$(CURDIR)/scripts:$$PATH" sha256sum axym > checksums.txt)

prepush-full: prepush codeql test-scenarios release-local
