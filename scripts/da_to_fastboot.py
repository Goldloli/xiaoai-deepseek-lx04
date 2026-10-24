# SPDX-License-Identifier: GPL-3.0-only
"""Check a live, cached DA session and request fastboot; never flash."""

import hashlib
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools/mtkclient"))
from mtkclient.config.mtk_config import MtkConfig
from mtkclient.Library.mtk_class import Mtk

config = MtkConfig()
config.reconnect = False
mtk = Mtk(config=config)
mtk.port.cdc.portconfig = {0x0E8D: {0x0003: -1, 0x2001: -1}}
if not mtk.port.wait_for_device(max_attempts=1, wait=False):
    raise SystemExit("No live DA device; enter download mode and run printgpt first.")
mtk.daloader.reinit()
if mtk.config.hwcode != 0x8167:
    raise SystemExit("Unexpected chip; aborting.")
data = mtk.daloader.readflash(0, 0x4400, "", "user", display=False)
with (ROOT / "backups/emmc-user.bin").open("rb") as backup:
    if data != backup.read(0x4400):
        raise SystemExit("GPT differs from backup; aborting.")
print("Live GPT matches backup:", hashlib.sha256(data).hexdigest(), flush=True)
print("Requesting temporary fastboot entry (no partition writes).", flush=True)
mtk.daloader.shutdown(bootmode=2)
