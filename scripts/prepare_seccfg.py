# SPDX-License-Identifier: GPL-3.0-only
"""Prepare and round-trip-check a local seccfg candidate using a live DA.

Only reads flash. Hardware cryptography uses temporary peripheral registers.
This script never calls writeflash, formatflash, or the automatic unlock method.
"""

import hashlib
import json
from pathlib import Path
import struct
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools/mtkclient"))
from mtkclient.config.mtk_config import MtkConfig
from mtkclient.Library.mtk_class import Mtk
from mtkclient.Library.Hardware.seccfg import SecCfgV4

config = MtkConfig()
config.reconnect = False
mtk = Mtk(config=config)
mtk.port.cdc.portconfig = {0x0E8D: {0x0003: -1, 0x2001: -1}}
if not mtk.port.wait_for_device(max_attempts=1, wait=False):
    raise SystemExit("No live DA session.")
mtk.daloader.reinit()
assert mtk.config.hwcode == 0x8167
raw = mtk.daloader.readflash(0x4C80000, 0x40000, "", "user", display=False)
assert raw == (ROOT / "backups/seccfg.bin").read_bytes(), "seccfg changed since backup"
para = mtk.daloader.readflash(0x6DC0000, 0x800, "", "user", display=False)
assert not para[:32].strip(b"\0"), "A pending boot command is present"
metadata = mtk.daloader.readflash(0x6DC0800, 0x200, "", "user", display=False)
assert struct.unpack_from("<I", metadata)[0] == 0x19191100
slots = {}
for i, slot in enumerate(("a", "b")):
    value = metadata[5+i]
    slots[slot] = dict(priority=value & 7, retry_count=(value >> 3) & 7,
                       successful_boot=bool(value & 64), normal_boot=bool(value & 128))
print("Live slot metadata:", json.dumps(slots), flush=True)

extension = mtk.daloader.xft or mtk.daloader.lft
assert extension is not None, "DA crypto extension is unavailable"
hwc = extension.cryptosetup()
original = SecCfgV4(hwc, mtk)
if not original.parse(raw):
    raise SystemExit("Original seccfg cryptographic validation failed; no candidate produced.")
assert original.lock_state == 1 and original.seccfg_size == 60
hwtype = original.hwtype
ok, block = original.create(lockflag="unlock", critical=False)
assert ok and len(block) == 512
candidate = bytes(block[:60]) + raw[60:]
assert struct.unpack_from("<I", candidate, 12)[0] == 3
assert candidate[:12] == raw[:12] and candidate[16:28] == raw[16:28]
assert candidate[60:] == raw[60:]
check = SecCfgV4(hwc, mtk)
assert check.parse(candidate) and check.lock_state == 3
ok, restored = check.create(lockflag="lock", critical=False)
assert ok and bytes(restored[:60]) == raw[:60], "Round-trip mismatch"
out = ROOT / "artifacts/seccfg-unlocked-candidate.bin"
if out.exists():
    raise FileExistsError(out)
out.write_bytes(candidate)
out.chmod(0o600)
manifest = dict(hwtype=hwtype, original_sha256=hashlib.sha256(raw).hexdigest(),
                candidate_sha256=hashlib.sha256(candidate).hexdigest(),
                size=len(candidate), round_trip_matches=True,
                original_lock_state=1, candidate_lock_state=3,
                dm_verity_unchanged=True, tail_preserved=True,
                slots=slots, flashed=False)
(ROOT / "artifacts/seccfg-candidate.json").write_text(json.dumps(manifest, indent=2)+"\n")
print("Local candidate prepared; cryptographic round trip matches. No flash writes.", flush=True)
mtk.port.close(reset=False)
