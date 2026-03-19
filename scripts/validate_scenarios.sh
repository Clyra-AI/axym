#!/usr/bin/env bash
set -euo pipefail

go test ./scenarios/... -count=1
go test ./internal/scenarios -count=1 -tags=scenario
