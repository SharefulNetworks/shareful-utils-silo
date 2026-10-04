#!/usr/bin/env bash
set -euo pipefail

export GOPROXY=https://proxy.golang.org,direct

if [ "$(id -u)" -ne 0 ]; then
  echo "reclaim_test.sh must run as root" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT_DIR/emptyshell"
LOGFILE="$(mktemp)"

cleanup() {
  rm -f "$LOGFILE"
}
trap cleanup EXIT

fail_with_log() {
  echo "$1" >&2
  if [ -f "$LOGFILE" ]; then
    echo "--- expect log ---" >&2
    cat "$LOGFILE" >&2 || true
    echo "--- end expect log ---" >&2
  fi
  exit 1
}

# 1) build emptyshell from the latest sources
cd "$ROOT_DIR"
go build -o "$BIN" ./cmd/emptyshell

# Gather root disk usage before the transient session is created.
before=$(df -B1 --output=used / | tail -1 | tr -d ' ')
echo "before=${before} bytes"

if ! command -v expect >/dev/null 2>&1; then
  echo "expect is required to automate the interactive shell" >&2
  echo "Install it with: apt-get install -y expect" >&2
  exit 1
fi

set +e
expect <<EOF 2>&1 | tee "$LOGFILE"
set timeout 300
log_user 1
log_file -a "$LOGFILE"
spawn "$BIN"
expect {
  "Entering EmptyShell..." { puts "banner seen" }
  timeout { puts stderr "timed out waiting for EmptyShell banner"; exit 1 }
}
expect {
  -re {silo#[^>]+> } { puts "shell prompt ready" }
  timeout { puts stderr "timed out waiting for silo prompt"; exit 1 }
}
send -- "apt-get update -qq\r"
expect {
  -re {silo#[^>]+> } { puts "apt update returned to shell prompt" }
  timeout { puts stderr "timed out waiting for apt-get update to finish"; exit 1 }
}
send -- "DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends mousepad\r"
expect {
  -re {silo#[^>]+> } { puts "install returned to shell prompt; sending exit" }
  timeout { puts stderr "timed out waiting for apt-get install to finish"; exit 1 }
}
send -- "exit\r"
expect eof
EOF
expect_status=${PIPESTATUS[0]}
set -e
if [ "$expect_status" -ne 0 ]; then
  fail_with_log "expect automation failed while exercising the transient shell"
fi

after=$(df -B1 --output=used / | tail -1 | tr -d ' ')
change=$((after - before))

printf 'after=%s bytes\n' "$after"
printf 'Disk usage change: %s bytes\n' "$change"

if mount | grep -q "emptyshell"; then
  echo "FAIL: stale emptyshell mounts still present" >&2
  mount | grep emptyshell >&2 || true
  exit 2
fi

if [ "$change" -le 0 ]; then
  echo "PASS: disk usage was not increased after transient shell exit"
  exit 0
fi

echo "FAIL: disk usage increased after transient shell exit" >&2
exit 1
