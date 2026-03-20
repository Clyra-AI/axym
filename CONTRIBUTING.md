# Contributing to Axym

Axym is an open-source Go CLI for deterministic AI governance evidence, proof records, compliance mapping, and audit-ready bundles.

This repository is licensed under [Apache-2.0](LICENSE). Unless explicitly stated otherwise, contributions are submitted under that same license.

## Before you start

- Keep work in scope for Axym only. Do not add Wrkr or Gait product features beyond Axym ingestion and interoperability contracts.
- Preserve determinism, offline-first defaults, fail-closed policy behavior, schema stability, and exit-code stability.
- Treat `--json`, output contracts, and release/install verification as public API surfaces.

## Local setup

Fast local path:

```bash
go build ./cmd/axym
./axym version --json
make lint-fast
make test-fast
make test-contracts
```

Normal contributors can usually stop at the fast local path unless they are changing public docs, release behavior, CI contracts, or other launch-facing surfaces.

Full local gate:

```bash
make prepush-full
```

Required tools for `make prepush-full`:

- `golangci-lint`
- `gosec`
- `codeql`

Maintainer and release-manager verification:

```bash
make release-local
make release-go-nogo-local
./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym
```

Additional required tools for `make release-local` and `make release-go-nogo-local`:

- `syft`
- `cosign`

## Change expectations

- Add or update tests at the right layer for every behavior change.
- Keep changes scoped to the story or bug you are addressing.
- Update user-facing docs in the same change when behavior, flags, schemas, or workflows change.
- Do not commit secrets, generated binaries, or transient reports.

## Pull requests

- Describe the user-visible change and any contract impact.
- Call out schema, CLI, exit-code, or workflow changes explicitly.
- Include the commands you ran and whether they passed.
- Flag any residual risk or deferred follow-up clearly.

Use the repo PR template for every change. Small doc-only fixes can be brief, but public-surface changes still need validation notes.

## Reporting bugs

Please include:

- Axym version and install path
- OS and Go version when relevant
- Exact command run
- Full machine-readable output when available
- Minimal fixture or repro steps

Open bug reports with the bug issue template so maintainers can reproduce the failure quickly.

## Feature requests

Use the feature request issue template for new ideas or workflow improvements. Frame the request around the operator problem first, then describe the proposed change and any determinism or offline-first constraints it must preserve.

## Maintainer and support expectations

- Maintainer support for the OSS CLI is best-effort and async. There is no guaranteed response SLA.
- Public GitHub issues are the default path for bugs, questions, and feature requests.
- Security-sensitive reports must use GitHub Security Advisories as the private reporting path described in [SECURITY.md](SECURITY.md).
- If GitHub Security Advisories are unavailable, open a minimal public issue without exploit details and reference [SECURITY.md](SECURITY.md).
- Maintainers may close or redirect requests that are out of scope for Axym, duplicate existing work, or require product commitments not available in the OSS CLI.
- Launch-facing docs and public issue threads should not promise private support channels, hosted services, or enterprise-only workflow commitments.

## Need help?

Use GitHub issues for normal bugs, questions, and feature requests. For security-sensitive reports, follow [SECURITY.md](SECURITY.md). Community behavior expectations live in [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
