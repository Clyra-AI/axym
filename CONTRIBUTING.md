# Contributing to Axym

Axym is an open-source Go CLI for deterministic AI governance evidence, proof records, compliance mapping, and audit-ready bundles.

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

## Reporting bugs

Please include:

- Axym version and install path
- OS and Go version when relevant
- Exact command run
- Full machine-readable output when available
- Minimal fixture or repro steps

## Need help?

Use GitHub issues for normal bugs, questions, and feature requests. For security-sensitive reports, follow [SECURITY.md](SECURITY.md).
