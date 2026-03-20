#!/usr/bin/env bash
set -euo pipefail

go test ./testinfra/contracts -run 'TestCommandSurfaceDocsParity|TestLaunchStoryDocsParity|TestOperatorDocsAreLinkedFromLaunchSurfaces|TestCommandInstallSurfaceDocsParity|TestLaunchDocsIndexReferencesSourceOfTruth|TestRepoHygiene' -count=1
