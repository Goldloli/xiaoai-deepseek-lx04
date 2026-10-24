#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Build only; never connects to USB or deploys to a speaker.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT="$ROOT/dist"
mkdir -p "$OUT"
cd "$ROOT/client"
for command in lx04-agent lx04-bootstrap lx04-netcheck lx04-searchcheck; do
    CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
        go build -trimpath -ldflags='-s -w' -o "$OUT/$command" "./cmd/$command"
done
python3 - "$OUT" <<'PY'
import hashlib, pathlib, sys
out = pathlib.Path(sys.argv[1])
names = ['lx04-agent', 'lx04-bootstrap', 'lx04-netcheck', 'lx04-searchcheck']
lines = [f'{hashlib.sha256((out / name).read_bytes()).hexdigest()}  {name}' for name in names]
(out / 'SHA256SUMS').write_text('\n'.join(lines) + '\n')
print('ARMv7 binaries and SHA256SUMS written to dist/. No device operations.')
PY
