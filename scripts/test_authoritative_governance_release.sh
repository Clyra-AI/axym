#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/axym-authoritative-release-test-XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
commit="0123456789abcdef0123456789abcdef01234567"
tag="v0.2.1-test"

grep -Fq 'gait_tag="${AXYM_GAIT_AUTHORITATIVE_RELEASE_TAG:-v1.7.2}"' "$root/.github/workflows/release.yml"
grep -Fq 'input_dir="${RUNNER_TEMP}/axym-authoritative-input"' "$root/.github/workflows/release.yml"
! grep -Fq '.tmp/authoritative-input' "$root/.github/workflows/release.yml"
grep -Fq -- '--gait-public-key "${AXYM_AUTHORITATIVE_GAIT_KEY_INPUT}"' "$root/.github/workflows/release.yml"
grep -Fq -- '--proposal-input "${AXYM_AUTHORITATIVE_PROPOSAL_INPUT}"' "$root/.github/workflows/release.yml"
grep -Fq "sha256sum >> checksums.txt" "$root/.github/workflows/release.yml"

go run ./scripts/generate_authoritative_governance_release.go \
  --out "$tmp_dir" --tag "$tag" --commit "$commit" \
  --proposal-input "testdata/action-contract-interop/v1/expected/compensation/pac-4b7f1402784256ce.json" \
  --activation-input "testdata/action-contract-interop/v1/expected/compensation/activated-action-contract.json" \
  --lifecycle-input "testdata/gait-action-contract-evidence/v1/compensation-required-started-completed/lifecycle.json" \
  --gait-public-key "testdata/gait-action-contract-evidence/v1/fixture-signing-key.public.b64" \
  --workflow-ref "Clyra-AI/axym/.github/workflows/release.yml@refs/tags/$tag" \
  --run-id test-run --repository Clyra-AI/axym
trusted_key="$(jq -r '.signing.public_key_sha256' "$tmp_dir/axym-authoritative-action-contract-manifest.json")"
go run ./scripts/verify_authoritative_governance_release.go \
  --root "$tmp_dir" --tag "$tag" --commit "$commit" --trusted-key-sha256 "$trusted_key"
jq -e '[.evidence[] | select(.kind == "compensation" and .attributes.state == "required")] | length > 0' "$tmp_dir/axym-authoritative-action-contract-evidence-packet.json" >/dev/null
jq -e '[.evidence[] | select(.kind == "compensation" and .attributes.state == "completed" and .attributes.terminal == "true")] | length > 0' "$tmp_dir/axym-authoritative-action-contract-evidence-packet.json" >/dev/null

if go run ./scripts/generate_authoritative_governance_release.go \
  --out "$tmp_dir/bad-gate-release" --tag "$tag" --commit "$commit" \
  --proposal-input "testdata/action-contract-interop/v1/expected/compensation/pac-4b7f1402784256ce.json" \
  --activation-input "testdata/action-contract-interop/v1/expected/compensation/activated-action-contract.json" \
  --lifecycle-input "testdata/gait-action-contract-evidence/v1/compensation-required-started-completed/lifecycle.json" \
  --gate-input "testdata/governance/v1/gait-gate/delegation-root.json" \
  --gait-public-key "testdata/gait-action-contract-evidence/v1/fixture-signing-key.public.b64" \
  --gate-public-key "testdata/governance/v1/gait-gate/fixture-signing-key.public.b64" \
  --workflow-ref "Clyra-AI/axym/.github/workflows/release.yml@refs/tags/$tag" \
  --run-id test-run --repository Clyra-AI/axym; then
  echo "mismatched gate contract digest was accepted" >&2
  exit 1
fi

cp "testdata/action-contract-interop/v1/expected/compensation/activated-action-contract.json" "$tmp_dir/bad-activation.json"
jq '.contract_id = "mismatched-contract"' "$tmp_dir/bad-activation.json" > "$tmp_dir/bad-activation.tmp"
mv "$tmp_dir/bad-activation.tmp" "$tmp_dir/bad-activation.json"
if go run ./scripts/generate_authoritative_governance_release.go \
  --out "$tmp_dir/bad-release" --tag "$tag" --commit "$commit" \
  --proposal-input "testdata/action-contract-interop/v1/expected/compensation/pac-4b7f1402784256ce.json" \
  --activation-input "$tmp_dir/bad-activation.json" \
  --lifecycle-input "testdata/gait-action-contract-evidence/v1/compensation-required-started-completed/lifecycle.json" \
  --gait-public-key "testdata/gait-action-contract-evidence/v1/fixture-signing-key.public.b64" \
  --workflow-ref "Clyra-AI/axym/.github/workflows/release.yml@refs/tags/$tag" \
  --run-id test-run --repository Clyra-AI/axym; then
  echo "mismatched or bad-signature activation was accepted" >&2
  exit 1
fi

cp "$tmp_dir/axym-authoritative-action-contract-manifest.json" "$tmp_dir/invalid.json"
jq '.fixture_only = true' "$tmp_dir/invalid.json" > "$tmp_dir/invalid.tmp"
mv "$tmp_dir/invalid.tmp" "$tmp_dir/axym-authoritative-action-contract-manifest.json"
if go run ./scripts/verify_authoritative_governance_release.go --root "$tmp_dir" --tag "$tag" --commit "$commit" --trusted-key-sha256 "$trusted_key"; then
  echo "fixture marker was accepted" >&2
  exit 1
fi

zeros="$(printf '%064d' 0)"
jq --arg bad "sha256:$zeros" '.signing.public_key_sha256 = $bad' "$tmp_dir/invalid.json" > "$tmp_dir/invalid.tmp"
mv "$tmp_dir/invalid.tmp" "$tmp_dir/axym-authoritative-action-contract-manifest.json"
if go run ./scripts/verify_authoritative_governance_release.go --root "$tmp_dir" --tag "$tag" --commit "$commit" --trusted-key-sha256 "$trusted_key"; then
  echo "public-key digest tamper was accepted" >&2
  exit 1
fi

jq '.schemas = []' "$tmp_dir/invalid.json" > "$tmp_dir/invalid.tmp"
mv "$tmp_dir/invalid.tmp" "$tmp_dir/axym-authoritative-action-contract-manifest.json"
if go run ./scripts/verify_authoritative_governance_release.go --root "$tmp_dir" --tag "$tag" --commit "$commit" --trusted-key-sha256 "$trusted_key"; then
  echo "incomplete schema manifest was accepted" >&2
  exit 1
fi

jq '.title = "tampered normative schema"' "$tmp_dir/axym-authoritative-action-contract-register.schema.json" > "$tmp_dir/schema.tmp"
mv "$tmp_dir/schema.tmp" "$tmp_dir/axym-authoritative-action-contract-register.schema.json"
schema_sha="sha256:$(shasum -a 256 "$tmp_dir/axym-authoritative-action-contract-register.schema.json" | awk '{print $1}')"
jq --arg digest "$schema_sha" '(.files[] | select(.path == "axym-authoritative-action-contract-register.schema.json")).sha256 = $digest | (.schemas[] | select(.path == "axym-authoritative-action-contract-register.schema.json")).sha256 = $digest' "$tmp_dir/invalid.json" > "$tmp_dir/invalid.tmp"
mv "$tmp_dir/invalid.tmp" "$tmp_dir/axym-authoritative-action-contract-manifest.json"
if go run ./scripts/verify_authoritative_governance_release.go --root "$tmp_dir" --tag "$tag" --commit "$commit" --trusted-key-sha256 "$trusted_key"; then
  echo "tampered normative schema was accepted" >&2
  exit 1
fi
