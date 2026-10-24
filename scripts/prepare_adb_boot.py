# SPDX-License-Identifier: GPL-3.0-only
"""Prepare LX04 boot candidates locally; never communicate with or flash a device."""

from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess


ROOT = Path(__file__).resolve().parents[1]
MAGISK = ROOT / "tools/magisk-host"
OUT = ROOT / "artifacts"


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def cpio_entries(data: bytes) -> dict[str, bytes]:
    entries = {}
    position = 0
    while position + 110 <= len(data):
        header = data[position : position + 110]
        if header[:6] not in (b"070701", b"070702"):
            raise ValueError("Unexpected CPIO header")
        fields = [int(header[6 + i * 8 : 14 + i * 8], 16) for i in range(13)]
        size, name_size = fields[6], fields[11]
        name = data[position + 110 : position + 110 + name_size - 1].decode()
        start = (position + 110 + name_size + 3) & ~3
        end = start + size
        if end > len(data):
            raise ValueError("Truncated CPIO entry")
        if name == "TRAILER!!!":
            return entries
        entries[name] = data[start:end]
        position = (end + 3) & ~3
    raise ValueError("CPIO trailer missing")


def run(args: list[str], cwd: Path, log) -> None:
    subprocess.run(args, cwd=cwd, stdout=log, stderr=subprocess.STDOUT, check=True)


def main() -> None:
    from dissect.extfs import ExtFS
    from dissect.util.stream import RangeStream

    manifest = json.loads((ROOT / "backups/critical-manifest.json").read_text())
    apk = ROOT / "tools/Magisk-v30.7.apk"
    assert sha256(apk) == "e0d32d2123532860f97123d927b1bb86c4e08e6fd8a48bfc6b5bee0afae9ebd5"
    host_key = (Path.home() / ".android/adbkey.pub").read_bytes().strip()
    assert len(host_key.split()[0]) > 500 and b"PRIVATE KEY" not in host_key
    with (ROOT / "backups/emmc-user.bin").open("rb") as source:
        fs = ExtFS(RangeStream(source, 0xD000000, 0x38000000))
        stock_keys = fs.get("/adb_keys").open().read().rstrip(b"\n")
    keys = stock_keys + b"\n"
    if host_key.split()[0] not in stock_keys:
        keys += host_key + b"\n"

    # Keep authentication enabled. Disable TCP ADB before starting USB ADB.
    rc = (
        "on post-fs-data\n"
        "    restorecon /adb_keys\n\n"
        "on property:sys.boot_completed=1\n"
        "    setprop service.adb.tcp.port -1\n"
        "    setprop persist.sys.usb.config accessory,adb\n"
        "    setprop sys.usb.config none\n"
        "    setprop sys.usb.config accessory,adb\n"
    ).encode()
    OUT.mkdir(mode=0o700, exist_ok=True)
    results = {}
    for slot in ("a", "b"):
        stock = ROOT / f"backups/boot_{slot}.bin"
        assert sha256(stock) == manifest[stock.name]["sha256"]
        assert stock.stat().st_size == 0x1000000
        work = ROOT / f"tools/boot-patch-{slot}"
        if work.exists():
            raise FileExistsError(f"Refusing to overwrite existing work directory: {work}")
        work.mkdir(mode=0o700)
        for name in ("magiskboot", "magiskinit", "magisk", "init-ld", "stub.apk", "boot_patch.sh"):
            shutil.copy2(MAGISK / name, work / name)
        (work / "magiskboot").chmod(0o755)
        with (ROOT / f"logs/prepare-boot-{slot}.log").open("wb") as log:
            run(["./magiskboot", "unpack", str(stock)], work, log)
            original_kernel = (work / "kernel").read_bytes()
            needle = b"skip_initramfs\0"
            replacement = b"want_initramfs\0"
            if original_kernel.count(needle) != 1:
                raise ValueError("Legacy system-as-root kernel marker is not unique")
            original_cpio = cpio_entries((work / "ramdisk.cpio").read_bytes())

            env = os.environ.copy()
            env.update(SOURCEDMODE="true", BOOTMODE="false", KEEPVERITY="true",
                       KEEPFORCEENCRYPT="true", PATCHVBMETAFLAG="false",
                       RECOVERYMODE="false", LEGACYSAR="true")
            wrapper = (
                'ui_print() { printf "%s\\n" "$*"; }; '
                'abort() { printf "%s\\n" "$*" >&2; exit 1; }; '
                '. ./boot_patch.sh "$1"'
            )
            subprocess.run(["/bin/bash", "-c", wrapper, "patch", str(stock)], cwd=work,
                           env=env, stdout=log, stderr=subprocess.STDOUT, check=True)
            (work / "lx04-adb.rc").write_bytes(rc)
            (work / "adb_keys").write_bytes(keys)
            run(["./magiskboot", "cpio", "ramdisk.cpio",
                 "add 0644 overlay.d/lx04-adb.rc lx04-adb.rc",
                 "add 0644 overlay.d/adb_keys adb_keys"], work, log)
            candidate = OUT / f"boot_{slot}-adb-candidate.img"
            run(["./magiskboot", "repack", str(stock), str(candidate)], work, log)
            if candidate.stat().st_size > stock.stat().st_size:
                raise ValueError("Patched image does not fit original boot partition")
            candidate.chmod(0o600)
            verify = work / "verify"
            verify.mkdir()
            run([str(work / "magiskboot"), "unpack", str(candidate)], verify, log)
            if (verify / "kernel").read_bytes() != original_kernel.replace(needle, replacement):
                raise ValueError("Unexpected kernel modifications")
            entries = cpio_entries((verify / "ramdisk.cpio").read_bytes())
            assert entries["overlay.d/lx04-adb.rc"] == rc
            assert entries["overlay.d/adb_keys"] == keys
            assert entries["init"] == (MAGISK / "magiskinit").read_bytes()
            assert entries["prop.default"] == original_cpio["prop.default"]
            assert b"ro.adb.secure=1\n" in entries["prop.default"]
            assert b"KEEPFORCEENCRYPT=true\n" in entries[".backup/.magisk"]
            assert b"KEEPVERITY=true\n" in entries[".backup/.magisk"]
            old, new = stock.read_bytes(), candidate.read_bytes()
            assert old[:8] == new[:8] == b"ANDROID!"
            for offset in (12, 20, 24, 28, 32, 36, 40, 44):
                assert struct.unpack_from("<I", old, offset) == struct.unpack_from("<I", new, offset)
            assert old[48:576] == new[48:576]
            results[slot] = {"stock_sha256": sha256(stock), "candidate_sha256": sha256(candidate),
                             "candidate_size": candidate.stat().st_size,
                             "checks": ["partition_fit", "boot_header_preserved", "only_legacy_sar_kernel_patch",
                                        "adb_auth_preserved", "original_adb_keys_preserved", "host_public_key_added",
                                        "encryption_preserved", "verity_preserved", "usb_only_adb_requested"],
                             "device_tested": False, "flashed": False}
        print(f"Slot {slot}: locally prepared and integrity checks passed ({candidate.stat().st_size} bytes)")
    (OUT / "boot-candidates.json").write_text(json.dumps(results, indent=2) + "\n")


if __name__ == "__main__":
    main()
