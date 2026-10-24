# SPDX-License-Identifier: GPL-3.0-only
"""Bounded LX04 boot trial. Default mode validates local files without USB.

--check-device reads flash only. --apply changes boot_b and one seccfg sector.
--restore writes the original boot_b and original seccfg sector.
Never formats, changes slots, or writes system/vendor/userdata/RPMB.
"""

import argparse
from datetime import datetime
import hashlib
import json
from pathlib import Path
import struct
import sys
import time

ROOT = Path(__file__).resolve().parents[1]
BOOT_B = 0x5DC0000
SECCFG = 0x4C80000
STOCK_HASHES = {
    "boot_a": "fc0904c26ff8e61363c4eff9c241bde5ae4ff7b49c3153e19b365d38a9f8a0c4",
    "boot_b": "abf6d12481188b4ae08b1ed2573c2f12f9efe7b9b3d6fc651d686be9cc65ddd4",
    "seccfg": "cfc2ba2c88cdc02565b27b1d98f2f2c7f90ceda58fe3f23518553fdad6097b33",
}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def local_files():
    stock = {name: (ROOT / f"backups/{name}.bin").read_bytes() for name in STOCK_HASHES}
    for name, expected in STOCK_HASHES.items():
        require(digest(stock[name]) == expected, f"Original {name} hash mismatch")
    patched = (ROOT / "artifacts/boot_b-adb-candidate.img").read_bytes()
    unlocked = (ROOT / "artifacts/seccfg-unlocked-candidate.bin").read_bytes()
    bm = json.loads((ROOT / "artifacts/boot-candidates.json").read_text())["b"]
    sm = json.loads((ROOT / "artifacts/seccfg-candidate.json").read_text())
    require(digest(patched) == bm["candidate_sha256"], "Boot candidate hash mismatch")
    require(len(patched) == len(stock["boot_b"]) == 0x1000000, "Unexpected boot size")
    require(digest(unlocked) == sm["candidate_sha256"], "Lock candidate hash mismatch")
    require(sm["round_trip_matches"] and sm["dm_verity_unchanged"], "Lock validation missing")
    require(len(unlocked) == len(stock["seccfg"]) == 0x40000, "Unexpected seccfg size")
    require(unlocked[:12] == stock["seccfg"][:12], "Unexpected lock header change")
    require(unlocked[16:28] == stock["seccfg"][16:28], "Unexpected verity/runtime change")
    require(unlocked[60:] == stock["seccfg"][60:], "Unexpected seccfg tail change")
    require(struct.unpack_from("<I", unlocked, 12)[0] == 3, "Expected unlock state 3")
    return stock, patched, unlocked


def connect():
    sys.path.insert(0, str(ROOT / "tools/mtkclient"))
    from mtkclient.config.mtk_config import MtkConfig
    from mtkclient.Library.mtk_class import Mtk

    config = MtkConfig()
    config.reconnect = False
    mtk = Mtk(config=config)
    mtk.port.cdc.portconfig = {0x0E8D: {0x0003: -1, 0x2001: -1}}
    require(mtk.port.wait_for_device(max_attempts=1, wait=False), "No live DA device")
    require(mtk.daloader.reinit(), "No cached DA state")
    require(mtk.config.hwcode == 0x8167, "Unexpected chip")
    before = read(mtk, 0, 0x4400)
    with (ROOT / "backups/emmc-user.bin").open("rb") as f:
        require(before == f.read(0x4400), "GPT differs from original backup")
    speed = mtk.daloader.da.get_usb_speed()
    if speed == b"full-speed":
        require(mtk.daloader.da.set_usb_speed(), "USB speed request failed")
        mtk.port.close(reset=True)
        for _ in range(30):
            if mtk.port.cdc.connect():
                break
            time.sleep(0.5)
        else:
            raise RuntimeError("USB speed reconnection failed; no flash writes made")
        require(mtk.daloader.da.get_usb_speed() == b"high-speed", "Unexpected USB speed")
        require(read(mtk, 0, 0x4400) == before, "GPT mismatch after USB reconnection")
    print("Connected; original GPT matches, USB:", mtk.daloader.da.get_usb_speed(), flush=True)
    return mtk


def read(mtk, address, size):
    data = mtk.daloader.readflash(address, size, "", "user", display=False)
    require(isinstance(data, (bytes, bytearray)) and len(data) == size, "Incomplete flash read")
    return bytes(data)


def write_and_verify(mtk, address, data):
    require((address == BOOT_B and len(data) == 0x1000000) or
            (address == SECCFG and len(data) == 0x200), "Write outside allowed regions")
    ok = mtk.daloader.writeflash(address, len(data), "", parttype="user", wdata=data, display=True)
    require(ok, f"Write failed at {address:#x}")
    require(read(mtk, address, len(data)) == data, f"Readback mismatch at {address:#x}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--check-device", action="store_true")
    group.add_argument("--apply", action="store_true")
    group.add_argument("--restore", action="store_true")
    args = parser.parse_args()
    stock, patched, unlocked = local_files()
    print("Local original/candidate hashes and write boundaries validated.", flush=True)
    if not any(vars(args).values()):
        return
    mtk = connect()
    try:
        live_a = read(mtk, 0x4DC0000, 0x1000000)
        live_b = read(mtk, BOOT_B, 0x1000000)
        live_cfg = read(mtk, SECCFG, 0x40000)
        require(live_a == stock["boot_a"], "Fallback boot_a differs; abort")
        if not args.restore:
            require(live_b == stock["boot_b"] and live_cfg == stock["seccfg"],
                    "Unexpected live boot/lock contents")
        para = read(mtk, 0x6DC0000, 0x1000)
        require(struct.unpack_from("<I", para, 0x800)[0] == 0x19191100, "Unknown slot format")
        if not args.restore:
            require(not para[:32].strip(b"\0"), "Pending boot command")
            require((para[0x806] & 7) > (para[0x805] & 7), "B is no longer preferred")
            require(bool(para[0x806] & 64), "B was not marked successful")
            print("Both original boot slots and lock match; slot B is preferred and successful.", flush=True)
        else:
            # A failed boot may change slot priorities or leave a partial image.
            # Preserve those observations locally; neither should prevent restoring
            # the two explicitly approved regions from verified original backups.
            stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
            snapshot = ROOT / "logs" / f"before-stock-restore-{stamp}"
            snapshot.mkdir(mode=0o700)
            for name, content in (("boot_b.bin", live_b), ("seccfg.bin", live_cfg), ("para.bin", para)):
                (snapshot / name).write_bytes(content)
            print("Restoration: fallback boot_a matches; saved current B/lock/slot contents.", flush=True)
        if args.check_device:
            return
        stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        journal = ROOT / f"logs/boot-trial-{stamp}.json"
        state = {"mode": "restore" if args.restore else "apply", "phase": "validated"}

        def record(phase):
            state["phase"] = phase
            journal.write_text(json.dumps(state, indent=2)+"\n")
            print(phase, flush=True)

        target_boot = stock["boot_b"] if args.restore else patched
        target_cfg = stock["seccfg"] if args.restore else unlocked
        record("starting boot_b write")
        try:
            write_and_verify(mtk, BOOT_B, target_boot)
            record("boot_b verified; writing seccfg sector")
            write_and_verify(mtk, SECCFG, target_cfg[:512])
            require(read(mtk, SECCFG, 0x40000) == target_cfg, "Full seccfg differs")
            record("all requested writes verified")
        except Exception:
            record("error; attempting stock restoration")
            write_and_verify(mtk, BOOT_B, stock["boot_b"])
            write_and_verify(mtk, SECCFG, stock["seccfg"][:512])
            require(read(mtk, SECCFG, 0x40000) == stock["seccfg"], "Restore check failed")
            record("stock restored and verified; no reboot requested")
            raise
        if args.apply:
            for name in ("boot-candidates.json", "seccfg-candidate.json"):
                path = ROOT / "artifacts" / name
                data = json.loads(path.read_text())
                entry = data["b"] if name.startswith("boot-") else data
                entry["flashed"] = True
                path.write_text(json.dumps(data, indent=2)+"\n")
        print("Requesting normal boot. A manual USB power cycle may be needed.", flush=True)
        require(mtk.daloader.shutdown(bootmode=1), "Normal boot request was not acknowledged")
    finally:
        mtk.port.close(reset=False)


if __name__ == "__main__":
    main()
