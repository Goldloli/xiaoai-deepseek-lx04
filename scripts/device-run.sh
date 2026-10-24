#!/system/bin/sh
# Runs entirely on the LX04. Credentials and logs stay in the private folder.
set -eu
umask 077
APP=/data/adb/lx04-deepseek
cd "$APP"
if [ -f disabled ]; then exit 0; fi
if [ -r agent.pid ]; then
    oldpid=$(cat agent.pid)
    case "$oldpid" in
        ''|*[!0-9]*) ;;
        *) if [ "$(readlink "/proc/$oldpid/exe" 2>/dev/null || true)" = "$APP/lx04-agent" ]; then exit 0; fi ;;
    esac
fi
until [ "$(getprop sys.boot_completed)" = 1 ]; do sleep 2; done
# Wait for Wi-Fi after boot; the app handles later connection interruptions.
attempt=0
until /system/bin/ip -4 route get 1.1.1.1 >/dev/null 2>&1; do
    attempt=$((attempt + 1))
    if [ "$attempt" -ge 60 ]; then exit 1; fi
    sleep 2
done
export HOME="$APP"
export SSL_CERT_DIR=/system/etc/security/cacerts
# Prefer an explicit, owner-only IP:port file. Otherwise try Android's DNS
# property, then the default gateway (which must provide DNS in that setup).
if [ -z "${XIAOBOT_DNS_SERVER:-}" ] && [ -r dns-server.txt ]; then
    XIAOBOT_DNS_SERVER=$(cat dns-server.txt)
fi
if [ -z "${XIAOBOT_DNS_SERVER:-}" ]; then
    dns_ip=$(/system/bin/getprop net.dns1)
    if [ -z "$dns_ip" ]; then
        while read -r destination via gateway rest; do
            if [ "$destination" = default ] && [ "$via" = via ]; then
                dns_ip=$gateway
                break
            fi
        done <<EOF
$(/system/bin/ip -4 route show default)
EOF
    fi
    [ -n "$dns_ip" ] || exit 1
    XIAOBOT_DNS_SERVER="$dns_ip:53"
fi
export XIAOBOT_DNS_SERVER
chmod 600 config.json .mi.token
if [ -f runtime.log ]; then mv -f runtime.log runtime.previous.log; fi
trap '"$APP/lx04-agent" -config "$APP/config.json" -restore-audio >> runtime.log 2>&1' EXIT
"$APP/lx04-agent" -config "$APP/config.json" > runtime.log 2>&1 &
agent_pid=$!
echo "$agent_pid" > agent.pid
trap 'kill "$agent_pid" 2>/dev/null || true' INT TERM
wait "$agent_pid"
