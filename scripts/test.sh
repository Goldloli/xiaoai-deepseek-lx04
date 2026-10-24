#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
python3 "$ROOT/scripts/check_repository.py"
for file in "$ROOT"/scripts/*.sh; do sh -n "$file"; done
cd "$ROOT/client"
go test . ./devicevoice ./runtimeenv ./jarvis/openai ./miservice ./deepseekweb ./voiceanswer \
    ./cmd/lx04-agent ./cmd/lx04-bootstrap ./cmd/lx04-netcheck ./cmd/lx04-searchcheck
