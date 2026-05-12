# Gait Contract Fixture Notes

This directory is reserved for Axym contract fixtures that model upstream Gait authorization and runtime-control artifacts.

Fixture ownership and provenance rules:

- Gait remains the source of truth for authorization bundles, freeze windows, kill switches, sandbox policy, explain output, and trust-graduation metadata.
- Axym fixtures in this repository are compatibility fixtures for deterministic parsing and verification tests.
- Update fixture payloads only when the corresponding Axym ingest or bundle contract changes, or when an upstream Gait contract change is being intentionally absorbed.
- Do not treat these fixtures as authoritative Gait release artifacts; they are Axym-side compatibility samples.
