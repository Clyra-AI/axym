#!/usr/bin/env bash
set -euo pipefail

dist_dir="dist"
binary_name="axym"
release_binary=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dist-dir)
      dist_dir="$2"
      shift 2
      ;;
    --binary-name)
      binary_name="$2"
      shift 2
      ;;
    --release-binary)
      release_binary="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 6
      ;;
  esac
done

fail() {
  echo "$1" >&2
  exit "${2:-1}"
}

require_file() {
  local path="$1"
  [[ -f "$path" ]] || fail "missing required file: $path" 7
}

source_smoke() {
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN
  go build -o "$tmp_dir/$binary_name" ./cmd/axym
  "$tmp_dir/$binary_name" version --json >/dev/null
}

release_binary_smoke() {
  local binary_path="$release_binary"
  if [[ -z "$binary_path" ]]; then
    binary_path="$dist_dir/$binary_name"
  fi
  require_file "$binary_path"
  chmod +x "$binary_path"
  "$binary_path" version --json >/dev/null
}

verify_checksums() {
  require_file "$dist_dir/checksums.txt"
  PATH="$(pwd)/scripts:$PATH" sha256sum -c "$dist_dir/checksums.txt" >/dev/null
}

verify_signature() {
  require_file "$dist_dir/checksums.txt"
  require_file "$dist_dir/checksums.txt.sig"
  if [[ -f "$dist_dir/checksums.txt.pem" ]]; then
    cosign verify-blob --certificate "$dist_dir/checksums.txt.pem" --signature "$dist_dir/checksums.txt.sig" "$dist_dir/checksums.txt" >/dev/null
    return
  fi
  if [[ -f "$dist_dir/local-cosign.pub" ]]; then
    cosign verify-blob --key "$dist_dir/local-cosign.pub" --signature "$dist_dir/checksums.txt.sig" --insecure-ignore-tlog=true "$dist_dir/checksums.txt" >/dev/null
    return
  fi
  fail "missing signature verification material in $dist_dir" 7
}

verify_sbom() {
  compgen -G "$dist_dir/*.spdx.json" >/dev/null || fail "missing SPDX SBOM in $dist_dir" 7
}

verify_provenance() {
  if compgen -G "$dist_dir/*.intoto.jsonl" >/dev/null; then
    return
  fi
  if compgen -G "$dist_dir/*provenance*.json" >/dev/null; then
    return
  fi
  fail "missing provenance receipt in $dist_dir" 7
}

verify_homebrew_contract() {
  grep -Fq "brews:" .goreleaser.yaml || fail "missing Homebrew packaging config" 3
  grep -Fq "brew install Clyra-AI/tap/axym" README.md || fail "README missing Homebrew install path" 3
  grep -Fq "brew install Clyra-AI/tap/axym" docs/commands/axym.md || fail "command guide missing Homebrew install path" 3
}

source_smoke
release_binary_smoke
verify_checksums
verify_signature
verify_sbom
verify_provenance
verify_homebrew_contract
