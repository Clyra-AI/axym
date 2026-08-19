# Action Contract fixture provenance

These are the nine unchanged Wrkr proposal artifact bytes from the released
fixture manifest pinned to producer `v1.14.0` (the v1.15 producer line keeps
the same artifact contract). The manifest is retained for IDs, revisions,
canonical content digests, and raw-byte SHA-256 verification; packet files are
intentionally not imported because this Axym slice consumes proposals only.

The Gait activation schema is copied from merged Gait commit `0897620`. That
commit ships the activation schema and deterministic activation tests, but no
standalone activation golden file; Axym's activation tests therefore generate
a producer-shaped signed artifact in memory and verify it against the copied
schema and exact proposal bytes.
