#!/system/bin/sh
# Runs entirely on the LX04. Credentials and logs stay in the private folder.
set -eu
umask 077
APP=/data/adb/lx04-deepseek
cd "$APP"
if [ -f disabled ]; then exit 0; fi
# The lock also covers boot/network waits and retry delays. An agent PID alone
# cannot prevent a second launcher from starting during those intervals.
exec 9> supervisor.lock
# Android's mksh marks exec-opened descriptors close-on-exec. An explicit
# duplicate makes the descriptor available to flock while the parent keeps it.
flock -n 3 3>&9 || exit 0
if [ -r agent.pid ]; then
    oldpid=$(cat agent.pid)
    case "$oldpid" in
        ''|*[!0-9]*) ;;
        *) if [ "$(readlink "/proc/$oldpid/exe" 2>/dev/null || true)" = "$APP/lx04-agent" ]; then exit 0; fi ;;
    esac
fi
echo "$$" > supervisor.pid
agent_pid=
cleanup() {
    if [ -n "$agent_pid" ]; then
        kill "$agent_pid" 2>/dev/null || true
        wait "$agent_pid" 2>/dev/null || true
        rm -f agent.pid
    fi
    "$APP/lx04-agent" -config "$APP/config.json" -restore-audio >> runtime.log 2>&1 || true
    rm -f supervisor.pid
}
trap cleanup EXIT
trap 'exit 0' INT TERM
log_status() {
    printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >> supervisor.log
    if [ "$(wc -c < supervisor.log)" -gt 65536 ]; then
        tail -c 32768 supervisor.log > supervisor.log.tmp
        mv -f supervisor.log.tmp supervisor.log
    fi
}
until [ "$(getprop sys.boot_completed)" = 1 ]; do
    [ ! -f disabled ] || exit 0
    sleep 2
done
export HOME="$APP"
export SSL_CERT_DIR=/system/etc/security/cacerts
configured_dns=${XIAOBOT_DNS_SERVER:-}
chmod 600 config.json
retry_delay=2
log_status 'supervisor started'
while [ ! -f disabled ]; do
    # A route to the router alone can appear before Android installs its
    # Internet policy route. Keep waiting if Wi-Fi comes up late.
    if ! /system/bin/ip -4 route get 1.1.1.1 >/dev/null 2>&1; then
        sleep 2
        continue
    fi
    # Re-read DNS after reconnecting. Explicit configuration takes precedence.
    XIAOBOT_DNS_SERVER=$configured_dns
    if [ -z "$XIAOBOT_DNS_SERVER" ] && [ -r dns-server.txt ]; then
        XIAOBOT_DNS_SERVER=$(cat dns-server.txt)
    fi
    if [ -z "$XIAOBOT_DNS_SERVER" ]; then
        dns_ip=$(/system/bin/getprop net.dns1)
        if [ -z "$dns_ip" ]; then
            read -r destination via gateway rest <<EOF
$(/system/bin/ip -4 route get 1.1.1.1)
EOF
            if [ "$via" = via ]; then dns_ip=$gateway; fi
        fi
        if [ -z "$dns_ip" ]; then
            log_status 'waiting for DNS configuration'
            sleep "$retry_delay"
            continue
        fi
        case "$dns_ip" in
            *:*) XIAOBOT_DNS_SERVER="[$dns_ip]:53" ;;
            *) XIAOBOT_DNS_SERVER="$dns_ip:53" ;;
        esac
    fi
    export XIAOBOT_DNS_SERVER
    if [ -f .mi.token ]; then chmod 600 .mi.token; fi
    if [ -f runtime.log ]; then mv -f runtime.log runtime.previous.log; fi
    started=$(date +%s)
    # Do not let the child or its logcat subprocess retain the supervisor lock.
    "$APP/lx04-agent" -config "$APP/config.json" >> runtime.log 2>&1 9>&- &
    agent_pid=$!
    echo "$agent_pid" > agent.pid
    log_status "agent started pid=$agent_pid"
    status=0
    wait "$agent_pid" || status=$?
    agent_pid=
    rm -f agent.pid
    "$APP/lx04-agent" -config "$APP/config.json" -restore-audio >> runtime.log 2>&1 || true
    [ ! -f disabled ] || break
    if [ "$(( $(date +%s) - started ))" -ge 60 ]; then retry_delay=2; fi
    log_status "agent exited status=$status; retry in ${retry_delay}s"
    sleep "$retry_delay"
    retry_delay=$((retry_delay * 2))
    if [ "$retry_delay" -gt 60 ]; then retry_delay=60; fi
done
