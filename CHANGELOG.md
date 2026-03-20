# Changelog

All notable changes to Axym will be documented in this file.

The format is based on Keep a Changelog, and Axym follows semver-style tagged releases for user-visible CLI and contract changes.

## Unreleased

### Added

- `init --sample-pack <dir>` for a supported offline first-value path that produces non-empty local evidence and compliance results.
- Launch-facing docs that separate `smoke test`, `sample proof path`, and `real integration path`.
- Root `LICENSE`, governance assets, issue templates, and PR template for the public OSS baseline.

### Changed

- Corrected the public first-value contract to `4` governance-event captures, `6` total sample records, `5/6` covered controls, grade `C`, and truthful `complete=false` / `weak_record_count=1` messaging.
- Corrected public install guidance so Homebrew users verify with `axym version --json` while source builds and unpacked release binaries use `./axym version --json`.
- Clarified contributor full-gate and release-local prerequisites, plus the explicit private and fallback public security reporting paths.
