# Contributing to Axym

Axym is an open-source Go CLI for deterministic AI governance evidence, proof records, compliance mapping, and audit-ready bundles.

This repository is licensed under [Apache-2.0](LICENSE). Unless explicitly stated otherwise, contributions are submitted under that same license.

## Before you start

- Keep work in scope for Axym only. Do not add Wrkr or Gait product features beyond Axym ingestion and interoperability contracts.
- Preserve determinism, offline-first defaults, fail-closed policy behavior, schema stability, and exit-code stability.
- Treat `--json`, output contracts, and release/install verification as public API surfaces.

## Local setup

```bash
go build ./cmd/axym
./axym version --json
make lint-fast
make test-fast
make test-contracts
```

For the full local gate:

```bash
make prepush-full
```

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
- Security-sensitive reports must not be filed publicly. Follow [SECURITY.md](SECURITY.md) instead.
- Maintainers may close or redirect requests that are out of scope for Axym, duplicate existing work, or require product commitments not available in the OSS CLI.
- Launch-facing docs and public issue threads should not promise private support channels, hosted services, or enterprise-only workflow commitments.

## Need help?

Use GitHub issues for normal bugs, questions, and feature requests. For security-sensitive reports, follow [SECURITY.md](SECURITY.md). Community behavior expectations live in [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
