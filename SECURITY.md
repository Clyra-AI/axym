# Security Policy

Axym is designed for deterministic, local-first evidence handling. Security bugs that could weaken proof integrity, schema validation, output safety, release verification, or secret handling should be reported privately before public disclosure.

## Supported security boundaries

Please report issues involving:

- proof record signing or chain verification bypass
- bundle integrity or provenance verification gaps
- fail-open behavior where Axym should fail closed
- schema validation bypass or unsafe artifact acceptance
- secret leakage, raw evidence exfiltration, or unsafe output path handling
- release artifact, checksum, signature, or SBOM verification issues

## How to report

- Prefer GitHub Security Advisories or another private maintainer channel before opening a public issue.
- If private reporting is not available, open a minimal public issue without exploit details and ask maintainers for a secure handoff path.

## What to include

- affected version or commit
- installation path used
- exact command or workflow involved
- minimal reproduction steps
- expected vs actual behavior
- impact on determinism, integrity, privacy, or policy enforcement

## Disclosure expectations

- Give maintainers a reasonable window to reproduce, patch, and publish guidance before full public disclosure.
- Avoid publishing raw exploit material that would expose real user evidence or secrets.

## Release verification

When validating a release-related report, include the verification path used, for example:

```bash
./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym
```

and any checksum, signature, or provenance output relevant to the issue.
