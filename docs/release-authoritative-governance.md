# Authoritative Action Contract release contract

The tag-triggered release workflow creates the release-owner Action Contract
governance handoff used by Proof final conformance. It is distinct from the
checked-in compatibility fixture under `testdata/governance/v1/producer-fixture`,
which remains synthetic, fixture-only, and quarantined.

For every release tag, the workflow generates a fresh Ed25519 key pair in the
runner, signs the producer-native register and evidence packet, and publishes:

- `axym-authoritative-action-contract-register.json`
- `axym-authoritative-action-contract-evidence-packet.json`
- `axym-authoritative-action-contract-manifest.json`
- `axym-authoritative-action-contract-public-key.b64`
- the two exact normative governance schemas
- `axym-authoritative-action-contract-bundle.tar.gz`

The manifest binds the release tag to its peeled commit, generator and workflow
identity, public-key digest, artifact digests, schema IDs/versions/digests, and
the explicit authority tuple `authoritative=true`, `fixture_only=false`,
`non_authoritative=false`, `quarantine=false`. The verifier rejects missing,
changed, fixture-marked, quarantined, unsigned, or relationship-inconsistent
artifacts.

Release checklist:

1. Run `make test-authoritative-release`.
2. Run the full pre-push, security, scenario, and documentation gates.
3. Generate and verify the release handoff with the tag and peeled commit.
4. Confirm the public key is release-generated and is not a checked-in fixture key.
5. Confirm the compressed bundle and standalone files are uploaded as release assets.
6. Confirm Proof imports the release manifest and verifies register, packet,
   signatures, schemas, relationships, and all digests offline.
