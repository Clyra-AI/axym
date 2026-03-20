#!/usr/bin/env bash
set -euo pipefail

go test ./testinfra/contracts -run 'TestLaunchStoryDocsSequence' -count=1
