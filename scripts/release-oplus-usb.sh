#!/bin/sh
# Temporarily pause only O+Connect's KeepAlive service, then restore it.
set -eu
duration=${1:-60}
case "$duration" in ''|*[!0-9]*) echo '暂停秒数必须是整数。' >&2; exit 1;; esac
if [ "$duration" -lt 1 ] || [ "$duration" -gt 600 ]; then
    echo '暂停时间必须在 1 到 600 秒之间。' >&2
    exit 1
fi
if [ "$(id -u)" != 0 ]; then
    echo '此脚本需要在 Mac 终端通过 sudo 运行。' >&2
    exit 1
fi
pid=$(/bin/launchctl print system/com.oplus.dsp.core | /usr/bin/awk '/^[[:space:]]*pid = / {print $3; exit}')
case "$pid" in ''|*[!0-9]*) echo '没有找到 O+Connect 后台服务，未做修改。' >&2; exit 1;; esac
actual=$(/bin/ps -p "$pid" -o comm=)
expected='/Applications/O+Connect.app/Contents/Resources/daemon/O+Connect Service'
if [ "$actual" != "$expected" ]; then
    echo '后台服务进程与预期不符，未做修改。' >&2
    exit 1
fi
resume() { /bin/kill -CONT "$pid" 2>/dev/null || true; }
trap resume EXIT HUP INT TERM
/bin/kill -STOP "$pid"
echo "O+Connect 后台服务已暂停，将在 $duration 秒后自动恢复。"
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$root/tools/platform-tools/adb" -P 5037 kill-server || true
/bin/sleep "$duration"
echo '即将恢复 O+Connect 后台服务。'
