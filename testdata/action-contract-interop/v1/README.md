# Action Contract fixture provenance

These are the nine unchanged Wrkr proposal artifact bytes from the released
fixture manifest pinned to producer `v1.14.0` (the v1.15 producer line keeps
the same artifact contract). The manifest is retained for IDs, revisions,
canonical content digests, and raw-byte SHA-256 verification; packet files are
intentionally not imported because this Axym slice consumes proposals only.

The Gait activation schema is copied from merged Gait commit `0897620`; the
exact activation fixture pack is released as Gait v1.4.0 at tag commit
`a44fdcf` and is traceable to the PR #170 fixture-generation commit
`4509fa7`. The released tag contains byte-identical manifest, key, and golden
assets. The pack contains six activated goldens and three explicit rejection
dispositions in `activation-fixture-manifest.json`, plus the public fixture
key. All fixture activations are marked
`development_signing: true`: the fixture-only option verifies raw schema,
binding, and signature encoding while remaining `unverifiable` and
non-authoritative by policy. Production/default verification rejects them.
