#!/usr/bin/env bash
# Validate go.work coverage and reject imports of deleted module paths.
# Compatible with macOS system bash 3.2 (no mapfile).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
cd "$ROOT"

fail=0

mods="$(find . -type d \( -name .git -o -name vendor -o -name node_modules -o -name embed_source -o -name .next -o -name docs \) -prune -o -name go.mod -print | sed 's|/go.mod||;s|^\./||' | sed 's|^$|.|' | sort -u)"
work="$(awk '/^use \(/,/^\)/ {
  for (i = 1; i <= NF; i++) {
    if ($i ~ /^\.\// || $i == ".") print $i
  }
}' go.work | sed 's|^\./||' | sort -u)"

mod_count=$(printf '%s\n' "$mods" | grep -c . || true)
work_count=$(printf '%s\n' "$work" | grep -c . || true)
printf 'Checking %s go.mod paths against go.work (%s entries)...\n' "$mod_count" "$work_count"

while IFS= read -r m; do
  [[ -n "$m" ]] || continue
  if ! printf '%s\n' "$work" | grep -Fx "$m" >/dev/null; then
    printf '  missing from go.work: %s\n' "$m" >&2
    fail=1
  fi
done <<EOF
$mods
EOF

printf 'Scanning Go imports for deleted module roots...\n'
# Exclude lingcli/fullmode.go (legacy path remapping table).
hits="$(grep -RIn --include='*.go' \
  -e '"github.com/LingByte/ling-base/mq"' \
  -e '"github.com/LingByte/ling-base/mq/' \
  -e '"github.com/LingByte/ling-base/agentkit"' \
  -e '"github.com/LingByte/ling-base/agentkit/' \
  -e '"github.com/LingByte/ling-base/voice/' \
  -e '"github.com/LingByte/ling-base/stores"' \
  -e '"github.com/LingByte/ling-base/stores/' \
  -e '"github.com/LingByte/ling-base/pentest"' \
  -e '"github.com/LingByte/ling-base/pentest/' \
  -e '"github.com/LingByte/ling-base/common/emoji"' \
  -e '"github.com/LingByte/ling-base/common/emoji/' \
  -e '"github.com/LingByte/ling-base/common/upload"' \
  -e '"github.com/LingByte/ling-base/common/upload/' \
  -e '"github.com/LingByte/ling-base/common/dnsutil"' \
  -e '"github.com/LingByte/ling-base/common/dnsutil/' \
  -e '"github.com/LingByte/ling-base/common/curlutil"' \
  -e '"github.com/LingByte/ling-base/common/curlutil/' \
  -e '"github.com/LingByte/ling-base/common/netutil/dnsutil"' \
  -e '"github.com/LingByte/ling-base/common/netutil/dnsutil/' \
  -e '"github.com/LingByte/ling-base/common/netutil/curlutil"' \
  -e '"github.com/LingByte/ling-base/common/netutil/curlutil/' \
  -e '"github.com/LingByte/ling-base/common/mathutil"' \
  -e '"github.com/LingByte/ling-base/common/mathutil/' \
  -e '"github.com/LingByte/ling-base/common/nltime"' \
  -e '"github.com/LingByte/ling-base/common/nltime/' \
  -e '"github.com/LingByte/ling-base/common/trie"' \
  -e '"github.com/LingByte/ling-base/common/trie/' \
  -e '"github.com/LingByte/ling-base/common/configcenter"' \
  -e '"github.com/LingByte/ling-base/common/configcenter/' \
  -e '"github.com/LingByte/ling-base/common/webhook"' \
  -e '"github.com/LingByte/ling-base/common/webhook/' \
  -e '"github.com/LingByte/ling-base/common/timeutil"' \
  -e '"github.com/LingByte/ling-base/common/timeutil/' \
  -e '"github.com/LingByte/ling-base/common/rbac"' \
  -e '"github.com/LingByte/ling-base/common/rbac/' \
  -e '"github.com/LingByte/ling-base/common/antispam"' \
  -e '"github.com/LingByte/ling-base/common/antispam/' \
  -e '"github.com/LingByte/ling-base/common/tree"' \
  -e '"github.com/LingByte/ling-base/common/tree/' \
  -e '"github.com/LingByte/ling-base/common/alerting"' \
  -e '"github.com/LingByte/ling-base/common/alerting/' \
  -e '"github.com/LingByte/ling-base/common/abtest"' \
  -e '"github.com/LingByte/ling-base/common/abtest/' \
  -e '"github.com/LingByte/ling-base/common/sse"' \
  -e '"github.com/LingByte/ling-base/common/sse/' \
  -e '"github.com/LingByte/ling-base/common/consistent"' \
  -e '"github.com/LingByte/ling-base/common/consistent/' \
  -e '"github.com/LingByte/ling-base/common/grpc"' \
  -e '"github.com/LingByte/ling-base/common/grpc/' \
  -e '"github.com/LingByte/ling-base/common/probe"' \
  -e '"github.com/LingByte/ling-base/common/probe/' \
  -e '"github.com/LingByte/ling-base/common/videoutil"' \
  -e '"github.com/LingByte/ling-base/common/videoutil/' \
  -e '"github.com/LingByte/ling-base/common/registry"' \
  -e '"github.com/LingByte/ling-base/common/registry/' \
  -e '"github.com/LingByte/ling-base/common/audioutil"' \
  -e '"github.com/LingByte/ling-base/common/audioutil/' \
  -e '"github.com/LingByte/ling-base/common/imageutil"' \
  -e '"github.com/LingByte/ling-base/common/imageutil/' \
  -e '"github.com/LingByte/ling-base/common/extension"' \
  -e '"github.com/LingByte/ling-base/common/extension/' \
  --exclude-dir=.git --exclude-dir=vendor --exclude-dir=node_modules \
  --exclude-dir=embed_source --exclude-dir=.next --exclude-dir=docs \
  . 2>/dev/null | grep -v 'lingcli/fullmode.go' || true)"

if [[ -n "$hits" ]]; then
  printf '%s\n' "$hits" | while IFS= read -r hit; do
    printf '  stale import: %s\n' "$hit" >&2
  done
  fail=1
fi

printf 'Scanning go.sum for deleted module roots...\n'
sum_hits="$(grep -RIn --include='go.sum' \
  -e 'github.com/LingByte/ling-base/mq' \
  -e 'github.com/LingByte/ling-base/agentkit' \
  -e 'github.com/LingByte/ling-base/voice/' \
  -e 'github.com/LingByte/ling-base/stores ' \
  -e 'github.com/LingByte/ling-base/stores/' \
  -e 'github.com/LingByte/ling-base/pentest ' \
  -e 'github.com/LingByte/ling-base/pentest/' \
  -e 'github.com/LingByte/ling-base/common/emoji' \
  -e 'github.com/LingByte/ling-base/common/upload' \
  -e 'github.com/LingByte/ling-base/common/dnsutil ' \
  -e 'github.com/LingByte/ling-base/common/dnsutil/' \
  -e 'github.com/LingByte/ling-base/common/curlutil ' \
  -e 'github.com/LingByte/ling-base/common/curlutil/' \
  -e 'github.com/LingByte/ling-base/common/netutil/dnsutil' \
  -e 'github.com/LingByte/ling-base/common/netutil/curlutil' \
  -e 'github.com/LingByte/ling-base/common/mathutil' \
  -e 'github.com/LingByte/ling-base/common/nltime' \
  -e 'github.com/LingByte/ling-base/common/trie' \
  -e 'github.com/LingByte/ling-base/common/configcenter' \
  -e 'github.com/LingByte/ling-base/common/webhook' \
  -e 'github.com/LingByte/ling-base/common/timeutil' \
  -e 'github.com/LingByte/ling-base/common/rbac' \
  -e 'github.com/LingByte/ling-base/common/antispam' \
  -e 'github.com/LingByte/ling-base/common/tree' \
  -e 'github.com/LingByte/ling-base/common/alerting' \
  -e 'github.com/LingByte/ling-base/common/abtest' \
  -e 'github.com/LingByte/ling-base/common/sse' \
  -e 'github.com/LingByte/ling-base/common/consistent' \
  -e 'github.com/LingByte/ling-base/common/grpc' \
  -e 'github.com/LingByte/ling-base/common/probe' \
  -e 'github.com/LingByte/ling-base/common/videoutil' \
  -e 'github.com/LingByte/ling-base/common/registry' \
  -e 'github.com/LingByte/ling-base/common/audioutil' \
  -e 'github.com/LingByte/ling-base/common/imageutil' \
  -e 'github.com/LingByte/ling-base/common/extension' \
  --exclude-dir=.git --exclude-dir=vendor --exclude-dir=node_modules \
  --exclude-dir=embed_source --exclude-dir=.next --exclude-dir=docs \
  . 2>/dev/null || true)"

if [[ -n "$sum_hits" ]]; then
  printf '%s\n' "$sum_hits" | while IFS= read -r hit; do
    printf '  stale go.sum entry: %s\n' "$hit" >&2
  done
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  printf 'check-modules failed\n' >&2
  exit 1
fi

printf 'check-modules OK\n'
