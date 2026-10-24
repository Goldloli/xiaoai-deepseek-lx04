# SPDX-License-Identifier: GPL-3.0-only
"""Prepare an earlier ADB setup trigger; --apply updates only the approved B slot."""

import argparse
import importlib.util
import json
from pathlib import Path
import shutil
import struct
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "scripts"))
from prepare_adb_boot import cpio_entries, sha256
import lx04_boot_trial as trial

IMAGE = ROOT / "artifacts/boot_b-adb-v2.img"
MANIFEST = ROOT / "artifacts/boot-b-adb-v2.json"
REVISION = 2
EXTRA_FILES = {}
RC = b"""on post-fs-data
    restorecon /adb_keys
    setprop service.adb.tcp.port -1
    setprop persist.sys.usb.config accessory,adb

on property:sys.boot_completed=1
    exec_background u:r:magisk:s0 root root -- /system/bin/settings put global adb_enabled 1
"""


def previous_image():
    if REVISION == 2:
        return ROOT / "artifacts/boot_b-adb-candidate.img"
    previous = REVISION - 1
    path = ROOT / f"artifacts/boot_b-adb-v{previous}.img"
    meta = json.loads((ROOT / f"artifacts/boot-b-adb-v{previous}.json").read_text())
    trial.require(sha256(path) == meta["candidate_sha256"], "Previous revision hash mismatch")
    return path


def prepare():
    stock, first, cfg = trial.local_files()
    old = previous_image()
    first = old.read_bytes()
    work = ROOT / f"tools/boot-patch-b-v{REVISION}"
    work.mkdir(mode=0o700)
    tool = ROOT / "tools/magisk-host/magiskboot"
    with (ROOT / f"logs/prepare-boot-b-v{REVISION}.log").open("wb") as log:
        def run(args, cwd=work):
            subprocess.run([str(tool), *args], cwd=cwd, stdout=log, stderr=subprocess.STDOUT, check=True)
        run(["unpack", str(old)])
        kernel = (work / "kernel").read_bytes()
        entries = cpio_entries((work / "ramdisk.cpio").read_bytes())
        (work / "lx04-adb.rc").write_bytes(RC)
        run(["cpio", "ramdisk.cpio", "add 0644 overlay.d/lx04-adb.rc lx04-adb.rc"])
        for name, content in EXTRA_FILES.items():
            (work / name).write_bytes(content)
            run(["cpio", "ramdisk.cpio", f"add 0644 overlay.d/sbin/{name} {name}"])
        run(["repack", str(old), str(IMAGE)])
        verify = work / "verify"
        verify.mkdir()
        run(["unpack", str(IMAGE)], cwd=verify)
        updated = cpio_entries((verify / "ramdisk.cpio").read_bytes())
        trial.require((verify / "kernel").read_bytes() == kernel, "Kernel changed")
        trial.require(set(updated) == set(entries) | {f"overlay.d/sbin/{n}" for n in EXTRA_FILES}, "Unexpected ramdisk paths")
        for name in entries:
            if name != "overlay.d/lx04-adb.rc":
                trial.require(entries[name] == updated[name], f"Unexpected ramdisk change: {name}")
        trial.require(updated["overlay.d/lx04-adb.rc"] == RC, "RC mismatch")
        for name, content in EXTRA_FILES.items():
            trial.require(updated[f"overlay.d/sbin/{name}"] == content, "Extra file mismatch")
        current = IMAGE.read_bytes()
        trial.require(len(current) == len(first) == 0x1000000, "Unexpected image size")
        for offset in (12, 20, 24, 28, 32, 36, 40, 44):
            trial.require(struct.unpack_from("<I", current, offset) == struct.unpack_from("<I", first, offset), "Boot header changed")
        trial.require(current[48:576] == first[48:576], "Boot command line changed")
    IMAGE.chmod(0o600)
    MANIFEST.write_text(json.dumps(dict(previous_sha256=trial.digest(first), candidate_sha256=sha256(IMAGE),
                                      changed_paths=["overlay.d/lx04-adb.rc"] + [f"overlay.d/sbin/{n}" for n in EXTRA_FILES], flashed=False,
                                      checks=["same_kernel", "same_other_ramdisk_entries", "same_boot_header",
                                              "same_adb_auth", "same_encryption_verity"]), indent=2)+"\n")
    print(f"V{REVISION} prepared; ADB init changes validated. No device communication.")


def apply():
    stock, first, cfg = trial.local_files()
    first = previous_image().read_bytes()
    meta = json.loads(MANIFEST.read_text())
    image = IMAGE.read_bytes()
    trial.require(trial.digest(image) == meta["candidate_sha256"], "V2 hash mismatch")
    trial.require(trial.digest(first) == meta["previous_sha256"], "V1 hash mismatch")
    mtk = trial.connect()
    try:
        trial.require(trial.read(mtk, 0x4DC0000, 0x1000000) == stock["boot_a"], "A slot changed")
        trial.require(trial.read(mtk, trial.BOOT_B, 0x1000000) == first, "B is not the tested V1")
        trial.require(trial.read(mtk, trial.SECCFG, 0x40000) == cfg, "Lock differs from candidate")
        para = trial.read(mtk, 0x6DC0000, 0x1000)
        trial.require(not para[:32].strip(b"\0"), "Pending boot command")
        trial.require(struct.unpack_from("<I", para, 0x800)[0] == 0x19191100 and
                      (para[0x806]&7) > (para[0x805]&7), "B is no longer preferred")
        try:
            trial.write_and_verify(mtk, trial.BOOT_B, image)
        except Exception:
            trial.write_and_verify(mtk, trial.BOOT_B, stock["boot_b"])
            trial.write_and_verify(mtk, trial.SECCFG, stock["seccfg"][:512])
            trial.require(trial.read(mtk, trial.SECCFG, 0x40000) == stock["seccfg"], "Restore check failed")
            raise
        meta["flashed"] = True
        MANIFEST.write_text(json.dumps(meta, indent=2)+"\n")
        print(f"V{REVISION} boot_b write and readback verified; seccfg and other partitions untouched.", flush=True)
        trial.require(mtk.daloader.shutdown(bootmode=1), "Normal boot request failed")
    finally:
        mtk.port.close(reset=False)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true")
    parser.add_argument("--revision", type=int, choices=(2, 3, 4, 5, 6), default=2)
    args = parser.parse_args()
    REVISION = args.revision
    if REVISION >= 3:
        IMAGE = ROOT / f"artifacts/boot_b-adb-v{REVISION}.img"
        MANIFEST = ROOT / f"artifacts/boot-b-adb-v{REVISION}.json"
        pub = (Path.home() / ".android/adbkey.pub").read_bytes().strip() + b"\n"
        trial.require(len(pub.split()[0]) > 500 and b"PRIVATE KEY" not in pub, "Invalid public key")
        EXTRA_FILES = {"lx04-adb.pub": pub, "lx04-adb-key.sh": b'''#!/system/bin/sh
set -eu
umask 077
key=$(cat "$1")
dir=/data/misc/adb
dest=$dir/adb_keys
mkdir -p "$dir"
[ ! -L "$dest" ] || exit 1
if [ ! -f "$dest" ] || ! grep -qF "$key" "$dest"; then
    printf '\\n%s\\n' "$key" >> "$dest"
fi
chown 1000:2000 "$dir" "$dest"
chmod 2750 "$dir"
chmod 0640 "$dest"
restorecon "$dir" "$dest"
'''}
        if REVISION == 3:
            RC = RC.replace(b"on post-fs-data\n", b"on post-fs-data\n    exec u:r:magisk:s0 root root -- /system/bin/sh ${MAGISKTMP}/lx04-adb-key.sh ${MAGISKTMP}/lx04-adb.pub\n", 1)
        elif REVISION == 4:
            # Native init builtins avoid executing a shell in the Magisk domain.
            # Preserve the previous data key file for immediate merging after ADB connects.
            commands = (b"on post-fs-data\n"
                b"    copy /data/misc/adb/adb_keys /data/misc/adb/adb_keys.lx04-before-v4\n"
                b"    mkdir /data/misc/adb 02750 system shell\n"
                b"    write /data/misc/adb/adb_keys " + pub.split()[0] + b"\n"
                b"    chown system shell /data/misc/adb/adb_keys\n"
                b"    chmod 0640 /data/misc/adb/adb_keys\n"
                b"    restorecon /data/misc/adb/adb_keys\n")
            RC = RC.replace(b"on post-fs-data\n", commands, 1)
            RC += (b"\non property:sys.usb.config=adb && property:sys.usb.configfs=1\n"
                   b"    write /config/usb_gadget/g1/bcdDevice 0x0401\n")
        else:
            from dissect.extfs import ExtFS
            from dissect.util.stream import RangeStream
            with (ROOT / "backups/emmc-user.bin").open("rb") as source:
                fs = ExtFS(RangeStream(source, 0xD000000, 0x38000000))
                keys = fs.get("/adb_keys").open().read().rstrip(b"\n") + b"\n"
            if pub.split()[0] not in keys:
                keys += pub
            EXTRA_FILES["lx04-adb-keys"] = keys
            if REVISION == 5:
                RC = RC.replace(b"on post-fs-data\n", b"on post-fs-data\n    mount none ${MAGISKTMP}/lx04-adb-keys /adb_keys bind\n", 1)
                RC += (b"\non property:sys.usb.config=adb && property:sys.usb.configfs=1\n"
                       b"    write /config/usb_gadget/g1/bcdDevice 0x0501\n")
            else:
                # A one-shot Magisk-domain service can mount/relabel the RAM key file
                # even when the stock init domain cannot mount over adb_keys_file.
                RC = RC.replace(b"on post-fs-data\n", b"on post-fs-data\n    start lx04-adb-authorize\n", 1)
                RC += b'''
service lx04-adb-authorize /system/bin/sh ${MAGISKTMP}/lx04-adb-auth.sh ${MAGISKTMP}/lx04-adb-keys
    user root
    group root shell
    seclabel u:r:magisk:s0
    disabled
    oneshot

on property:sys.usb.config=adb && property:sys.usb.configfs=1
    write /config/usb_gadget/g1/bcdDevice 0x0600
'''
                EXTRA_FILES["lx04-adb-auth.sh"] = b'''#!/system/bin/sh
status=0x0601
src=$1
if [ ! -r "$src" ]; then
    status=0x06e1
elif ! /system/bin/mount -o bind "$src" /adb_keys; then
    status=0x06e2
else
    /system/bin/restorecon /adb_keys
    if ! /system/bin/grep -qF "''' + pub.split()[0] + b'''" /adb_keys; then
        status=0x06e3
    fi
fi
# Wait for USB and Android's last USB-setting update, then enumerate the
# diagnostic revision. No authentication settings or data files are changed.
i=0
while [ "$i" -lt 90 ]; do
    [ "$(/system/bin/getprop sys.boot_completed)" = 1 ] && break
    /system/bin/sleep 1
    i=$((i+1))
done
/system/bin/sleep 3
g=/config/usb_gadget/g1
if [ -e "$g/bcdDevice" ]; then
    printf '%s' "$status" > "$g/bcdDevice"
    printf 'none' > "$g/UDC"
    /system/bin/sleep 1
    /system/bin/getprop sys.usb.controller > "$g/UDC"
fi
'''
    apply() if args.apply else prepare()
