SHELL := /bin/bash

.PHONY: lint-fast lint-go test-fast test-contracts test-scenarios test-hardening test-chaos test-perf test-security test-adapter-parity test-docs-consistency test-docs-storyline test-docs-links prepush codeql release-local release-go-nogo-local prepush-full

lint-fast:
	@test -z "$$(gofmt -l .)" || (echo "gofmt required"; gofmt -l .; exit 1)
	@go vet ./...

lint-go:
	@command -v golangci-lint >/dev/null || (echo "golangci-lint is required"; exit 7)
	@golangci-lint run ./...

test-fast:
	@go test ./... -count=1

test-contracts:
	@go test ./testinfra/contracts/... -count=1

test-scenarios:
	@./scripts/validate_scenarios.sh

test-hardening:
	@go test ./... -count=1 -run TestHardening

test-chaos:
	@go test ./... -count=1 -run TestChaos

test-perf:
	@go test ./... -count=1 -run TestPerf

test-security:
	@command -v gosec >/dev/null || (echo "gosec is required"; exit 7)
	@gosec ./...

test-adapter-parity:
	@go test ./... -count=1 -run TestAdapterParity

test-docs-consistency:
	@./scripts/check_docs_consistency.sh

test-docs-storyline:
	@./scripts/check_docs_storyline.sh

test-docs-links:
	@./scripts/check_docs_links.sh

prepush: lint-fast test-fast test-contracts

codeql:
	@command -v codeql >/dev/null || (echo "codeql CLI is required"; exit 7)
	@rm -rf .codeql
	@mkdir -p .codeql
	@codeql database create .codeql/db --language=go --source-root=. --command='go build ./cmd/axym'
	@codeql database analyze .codeql/db codeql/go-queries:codeql-suites/go-security-and-quality.qls --format=sarif-latest --output .codeql/results.sarif

release-local:
	@rm -rf dist
	@mkdir -p dist
	@go build -o dist/axym ./cmd/axym
	@(cd dist && PATH="$(CURDIR)/scripts:$$PATH" sha256sum axym > checksums.txt)
	@syft scan dist/axym -o spdx-json=dist/axym.spdx.json >/dev/null
	@rm -f dist/local-cosign.key dist/local-cosign.pub dist/checksums.txt.sig dist/checksums.txt.pem dist/axym.intoto.jsonl
	@COSIGN_PASSWORD="" cosign generate-key-pair --output-key-prefix dist/local-cosign >/dev/null 2>&1
	@COSIGN_PASSWORD="" cosign sign-blob --yes --tlog-upload=false --key dist/local-cosign.key --output-signature dist/checksums.txt.sig dist/checksums.txt >/dev/null
	@binary_sha="$$(PATH="$(CURDIR)/scripts:$$PATH" sha256sum dist/axym | awk '{print $$1}')"; \
	printf '{"_type":"https://in-toto.io/Statement/v0.1","subject":[{"name":"axym","digest":{"sha256":"%s"}}],"predicateType":"https://slsa.dev/provenance/v1","predicate":{"buildType":"axym-local-release","builder":{"id":"local-release"}}}\n' "$$binary_sha" > dist/axym.intoto.jsonl

release-go-nogo-local:
	@./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym

prepush-full: prepush lint-go test-security codeql test-scenarios test-docs-consistency test-docs-storyline test-docs-links release-local release-go-nogo-local
