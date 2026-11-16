#!/usr/bin/env bash
# Release ling-base modules using their current go.mod paths.
# New module paths start at v0.1.0; historical tags are never moved or reused.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
cd "$ROOT"
PKG=""
GROUP=""
LEVEL="patch"
ALL=false
DRY_RUN=false
LIST_TAGS=false
MODULE_ROOT="github.com/LingByte/ling-base"

usage() {
  printf '%s\n' \
    'Usage: scripts/release.sh --pkg <dir> [--level patch|minor|major] [--dry-run]' \
    '       scripts/release.sh --group <name> [--level patch|minor|major] [--dry-run]' \
    '       scripts/release.sh --all [--level patch|minor|major] [--dry-run]' \
    '       scripts/release.sh --pkg <dir> --list-tags' \
    '' \
    'Prefer --pkg or --group for routine releases; --all scans every go.mod (~238 modules).' \
    'Groups: relay | common-infra | voice | providers-other | tooling | common' \
    'Examples: --pkg common/stores, --group voice, --pkg lingcli' \
    '--all / --group release libraries; release lingcli separately after publishing libraries.' \
    'This script creates local tags only and requires a clean working tree.'
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pkg|--level|--group)
      [[ $# -ge 2 && "$2" != --* ]] || { printf 'Missing value for %s\n' "$1" >&2; exit 1; }
      if [[ "$1" == --pkg ]]; then PKG="$2"
      elif [[ "$1" == --group ]]; then GROUP="$2"
      else LEVEL="$2"; fi
      shift 2 ;;
    --all) ALL=true; shift ;;
    --dry-run) DRY_RUN=true; shift ;;
    --list-tags) LIST_TAGS=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'Unknown option: %s\n' "$1" >&2; exit 1 ;;
  esac
done

[[ "$LEVEL" == patch || "$LEVEL" == minor || "$LEVEL" == major ]] || { printf 'Invalid release level: %s\n' "$LEVEL" >&2; exit 1; }
mode_count=0
[[ "$ALL" == true ]] && mode_count=$((mode_count + 1))
[[ -n "$PKG" ]] && mode_count=$((mode_count + 1))
[[ -n "$GROUP" ]] && mode_count=$((mode_count + 1))
if [[ "$mode_count" -gt 1 ]]; then
  printf '%s\n' 'Use only one of --all, --pkg, or --group' >&2
  exit 1
fi
if [[ "$LIST_TAGS" == true && ( "$ALL" == true || -n "$GROUP" ) ]]; then
  printf '%s\n' '--list-tags requires --pkg' >&2
  exit 1
fi

# Print go.mod directories matching a release group (excludes lingcli).
list_group_dirs() {
  local name="$1" dir
  while IFS= read -r manifest; do
    dir="${manifest%/go.mod}"
    dir="${dir#./}"
    [[ -n "$dir" ]] || dir=.
    [[ "$dir" != lingcli ]] || continue
    case "$name" in
      relay)
        [[ "$dir" == relay || "$dir" == relay/* ]] || continue ;;
      voice)
        [[ "$dir" == providers/voice || "$dir" == providers/voice/* ]] || continue ;;
      providers-other)
        [[ "$dir" == providers/ocr || "$dir" == providers/ocr/* || "$dir" == providers/censor || "$dir" == providers/censor/* ]] || continue ;;
      tooling)
        [[ "$dir" == bootstrap || "$dir" == version || "$dir" == apidocs || "$dir" == toolkits || "$dir" == toolkits/* ]] || continue ;;
      common-infra)
        case "$dir" in
          common/limiter|common/limiter/*|common/circuitbreaker|common/retry|common/cache|common/cache/*|common/lock|common/lock/*|common/mq|common/mq/*|common/queue|common/queue/*|common/bloom|common/bloom/*|common/pool|common/eventbus|common/scheduler) ;;
          *) continue ;;
        esac ;;
      common)
        [[ "$dir" == common || "$dir" == common/* ]] || continue ;;
      *)
        printf 'Unknown group: %s (relay|common-infra|voice|providers-other|tooling|common)\n' "$name" >&2
        return 1 ;;
    esac
    printf '%s\n' "$dir"
  done < <(find . -type d \( -name .git -o -name vendor -o -name node_modules -o -name embed_source -o -name .next \) -prune -o -name go.mod -print | sort)
}

normalize_dir() {
  local dir="$1" resolved
  case "${dir#./}" in
    stores|stores/*) printf 'stores moved to common/stores; use --pkg common/%s\n' "${dir#./}" >&2; return 1 ;;
    voice|voice/*) printf 'voice moved to providers/voice; use --pkg providers/%s\n' "${dir#./}" >&2; return 1 ;;
    pentest|pentest/*) printf 'pentest moved to toolkits/pentest; use --pkg toolkits/pentest\n' >&2; return 1 ;;
    common/dnsutil|common/dnsutil/*|common/netutil/dnsutil|common/netutil/dnsutil/*) printf 'dnsutil folded into common/netutil; use --pkg common/netutil\n' >&2; return 1 ;;
    common/curlutil|common/curlutil/*|common/netutil/curlutil|common/netutil/curlutil/*) printf 'curlutil folded into common/netutil; use --pkg common/netutil\n' >&2; return 1 ;;
    common/emoji|common/emoji/*) printf 'common/emoji was removed\n' >&2; return 1 ;;
    common/upload|common/upload/*) printf 'common/upload was removed; use common/stores\n' >&2; return 1 ;;
    common/mathutil|common/mathutil/*) printf 'common/mathutil was removed; use github.com/LingByte/ling-base/common helpers\n' >&2; return 1 ;;
    common/nltime|common/nltime/*) printf 'common/nltime was folded into package common (ParseNaturalTime); use github.com/LingByte/ling-base/common\n' >&2; return 1 ;;
    common/trie|common/trie/*) printf 'common/trie was removed\n' >&2; return 1 ;;
    common/configcenter|common/configcenter/*) printf 'common/configcenter was removed\n' >&2; return 1 ;;
    common/webhook|common/webhook/*) printf 'common/webhook was removed; use common/notification/webhook\n' >&2; return 1 ;;
    common/timeutil|common/timeutil/*) printf 'common/timeutil was folded into package common; use github.com/LingByte/ling-base/common\n' >&2; return 1 ;;
    common/rbac|common/rbac/*) printf 'common/rbac was removed\n' >&2; return 1 ;;
    common/antispam|common/antispam/*) printf 'common/antispam was removed\n' >&2; return 1 ;;
    common/tree|common/tree/*) printf 'common/tree was removed\n' >&2; return 1 ;;
    common/alerting|common/alerting/*) printf 'common/alerting was removed\n' >&2; return 1 ;;
    common/abtest|common/abtest/*) printf 'common/abtest was removed\n' >&2; return 1 ;;
    common/sse|common/sse/*) printf 'common/sse was removed\n' >&2; return 1 ;;
    common/consistent|common/consistent/*) printf 'common/consistent was removed\n' >&2; return 1 ;;
    common/grpc|common/grpc/*) printf 'common/grpc was removed\n' >&2; return 1 ;;
    common/probe|common/probe/*) printf 'common/probe was folded into common/netutil; use github.com/LingByte/ling-base/common/netutil\n' >&2; return 1 ;;
    common/videoutil|common/videoutil/*) printf 'common/videoutil moved to common/media/video; use --pkg common/media/video\n' >&2; return 1 ;;
    common/registry|common/registry/*) printf 'common/registry was removed\n' >&2; return 1 ;;
    common/audioutil|common/audioutil/*) printf 'common/audioutil moved to common/media/audio; use --pkg common/media/audio\n' >&2; return 1 ;;
    common/imageutil|common/imageutil/*) printf 'common/imageutil moved to common/media/image; use --pkg common/media/image\n' >&2; return 1 ;;
    common/extension|common/extension/*) printf 'common/extension was removed\n' >&2; return 1 ;;
  esac
  resolved=$(cd "$dir" 2>/dev/null && pwd -P) || { printf 'Directory not found: %s\n' "$dir" >&2; return 1; }
  if [[ "$resolved" == "$ROOT" ]]; then
    printf '.\n'
  elif [[ "$resolved" == "$ROOT/"* ]]; then
    printf '%s\n' "${resolved#"$ROOT/"}"
  else
    printf 'Module directory must be inside this repository: %s\n' "$dir" >&2
    return 1
  fi
}

get_module_path() {
  local key value rest
  [[ -f "$1/go.mod" ]] || { printf 'No go.mod in %s\n' "$1" >&2; return 1; }
  while read -r key value rest; do
    if [[ "$key" == module ]]; then printf '%s\n' "${value//\"/}"; return; fi
  done < "$1/go.mod"
  printf 'Missing module directive in %s/go.mod\n' "$1" >&2
  return 1
}

get_prefix() {
  local dir="$1" module expected="$MODULE_ROOT"
  [[ "$dir" == . ]] || expected="$MODULE_ROOT/$dir"
  module=$(get_module_path "$dir") || return 1
  [[ "$module" == "$expected" ]] || { printf 'Module path %s does not match directory %s\n' "$module" "$dir" >&2; return 1; }
  if [[ "$dir" != . ]]; then printf '%s/\n' "${module#"$MODULE_ROOT/"}"; fi
}

# Match only this module's stable tags, not descendant modules or prereleases.
list_module_tags() {
  local prefix tag version
  prefix=$(get_prefix "$1") || return 1
  while IFS= read -r tag; do
    version="${tag#"$prefix"}"
    if [[ "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
      printf '%s\n' "$tag"
    fi
  done < <(git tag --list "${prefix}v*" --sort=-v:refname)
}

get_latest_tag() {
  local tag
  # Read the entire stream to avoid SIGPIPE under set -o pipefail.
  local first=""
  while IFS= read -r tag; do [[ -n "$first" ]] || first="$tag"; done < <(list_module_tags "$1")
  printf '%s\n' "$first"
}

bump_version() {
  local major minor patch
  IFS=. read -r major minor patch <<< "$1"
  case "$2" in
    patch) patch=$((patch + 1)) ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    major) major=$((major + 1)); minor=0; patch=0 ;;
  esac
  printf '%s.%s.%s\n' "$major" "$minor" "$patch"
}

has_changes() {
  local tag file
  tag=$(get_latest_tag "$1")
  if [[ -z "$tag" ]]; then return 0; fi
  while IFS= read -r file; do
    if [[ "$file" != *_test.go ]]; then return 0; fi
  done < <(git diff --name-only "$tag" -- "$1")
  return 1
}

release_module() {
  local dir="$1" prefix tag next version
  prefix=$(get_prefix "$dir") || return 1
  tag=$(get_latest_tag "$dir")
  if [[ -z "$tag" ]]; then
    next="${prefix}v0.1.0"
  else
    version="${tag#"${prefix}v"}"
    next="${prefix}v$(bump_version "$version" "$LEVEL")"
  fi
  if [[ "$DRY_RUN" == true ]]; then
    printf '  [dry-run] %s (%s)\n' "$next" "$(get_module_path "$dir")"
  else
    git tag -a "$next" -m "Release $next — $(get_module_path "$dir")"
    printf '  Tagged %s\n' "$next"
  fi
}

if [[ "$ALL" != true && -z "$GROUP" ]]; then
  [[ -n "$PKG" ]] || { usage >&2; exit 1; }
  PKG=$(normalize_dir "$PKG")
  get_prefix "$PKG" >/dev/null
  if [[ "$LIST_TAGS" == true ]]; then list_module_tags "$PKG"; exit 0; fi
fi

if [[ "$DRY_RUN" != true ]]; then
  if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
    printf '%s\n' 'Commit all release changes first; refusing to tag an uncommitted working tree. Use --dry-run to preview.' >&2
    exit 1
  fi
  if [[ "$PKG" == lingcli ]]; then
    go run ./scripts/moduleversions --check --require-tags
  fi
fi

release_dirs() {
  local count=0 dir
  while IFS= read -r dir; do
    [[ -n "$dir" ]] || continue
    if has_changes "$dir"; then release_module "$dir"; count=$((count + 1)); fi
  done
  printf '%s module(s). Publish library tags, then run make sync-cli-versions, commit, and release lingcli separately.\n' "$count"
}

if [[ "$ALL" == true ]]; then
  {
    while IFS= read -r manifest; do
      dir="${manifest%/go.mod}"
      dir="${dir#./}"
      [[ -n "$dir" ]] || dir=.
      # The CLI must be built with the versions of already published libraries.
      [[ "$dir" != lingcli ]] || continue
      printf '%s\n' "$dir"
    done < <(find . -type d \( -name .git -o -name vendor -o -name node_modules -o -name embed_source -o -name .next \) -prune -o -name go.mod -print | sort)
  } | release_dirs
elif [[ -n "$GROUP" ]]; then
  list_group_dirs "$GROUP" | release_dirs
else
  release_module "$PKG"
fi
