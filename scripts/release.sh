#!/usr/bin/env bash
# Release script for ling-base multi-module Go library.
#
# Usage:
#   scripts/release.sh --pkg <dir> --level <patch|minor|major>
#   scripts/release.sh --all --level <patch|minor|major>
#
# Examples:
#   scripts/release.sh --pkg scheduler --level patch
#     # scheduler/v0.1.0 → scheduler/v0.1.1
#
#   scripts/release.sh --pkg common/jwtutil --level minor
#     # common/jwtutil/v0.1.0 → common/jwtutil/v0.2.0
#
#   scripts/release.sh --pkg . --level patch
#     # v0.2.1 → v0.2.2 (root module)
#
#   scripts/release.sh --all --level patch
#     # Auto-detect all modules with changes since their last tag
#     # and bump each one by the specified level.
#
# What it does:
#   1. Finds the latest existing tag for the module (e.g. scheduler/v0.1.0).
#   2. Increments the version according to --level.
#   3. Creates an annotated git tag at HEAD.
#   4. Prints a summary.
#
# This script does NOT push tags. Use `make push-tags` or `git push origin --tags`.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# ──────────────────────────────────────────────
# Argument parsing
# ──────────────────────────────────────────────

PKG=""
LEVEL="patch"
ALL=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --pkg)
      PKG="$2"
      shift 2
      ;;
    --level)
      LEVEL="$2"
      shift 2
      ;;
    --all)
      ALL=true
      shift
      ;;
    -h|--help)
      sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [[ "$LEVEL" != "patch" && "$LEVEL" != "minor" && "$LEVEL" != "major" ]]; then
  echo "Error: --level must be one of: patch, minor, major" >&2
  exit 1
fi

# ──────────────────────────────────────────────
# Functions
# ──────────────────────────────────────────────

# Get the tag prefix for a directory.
# Root dir (.) → "" (tags are v0.1.0, v0.2.0, ...)
# Subdir (scheduler) → "scheduler/" (tags are scheduler/v0.1.0, ...)
get_prefix() {
  local dir="$1"
  if [[ "$dir" == "." || -z "$dir" ]]; then
    echo ""
  else
    echo "${dir}/"
  fi
}

# Get the latest tag for a module directory.
# Returns empty string if no tag exists.
get_latest_tag() {
  local dir="$1"
  local prefix
  prefix=$(get_prefix "$dir")

  if [[ -z "$prefix" ]]; then
    # Root module: match v* but not */v*
    git tag --list "v[0-9]*" --sort=-v:refname | head -1
  else
    git tag --list "${prefix}v*" --sort=-v:refname | head -1
  fi
}

# Bump a version string (X.Y.Z) by the given level.
# Usage: bump_version "0.1.0" patch → "0.1.1"
bump_version() {
  local ver="$1"
  local level="$2"

  local major minor patch
  IFS='.' read -r major minor patch <<< "$ver"

  case "$level" in
    patch)
      patch=$((patch + 1))
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
  esac

  echo "${major}.${minor}.${patch}"
}

# Check if a module directory has changes since its last tag.
# Returns 0 (true) if there are changes, 1 (false) otherwise.
has_changes() {
  local dir="$1"
  local last_tag
  last_tag=$(get_latest_tag "$dir")

  if [[ -z "$last_tag" ]]; then
    # No tag ever — check if dir has any .go files (excluding tests).
    if find "$dir" -maxdepth 1 -name '*.go' ! -name '*_test.go' 2>/dev/null | grep -q .; then
      return 0
    fi
    return 1
  fi

  # Check for non-test .go file changes, or go.mod changes.
  local changes
  changes=$(git diff --name-only "${last_tag}..HEAD" -- "$dir" 2>/dev/null | grep -v '_test\.go$' | head -1)
  if [[ -n "$changes" ]]; then
    return 0
  fi
  return 1
}

# Get the module path from go.mod.
get_module_path() {
  local dir="$1"
  head -1 "${dir}/go.mod" | awk '{print $2}'
}

# Release a single module: bump version and create tag.
release_module() {
  local dir="$1"
  local level="$2"

  local prefix last_tag mod
  prefix=$(get_prefix "$dir")
  last_tag=$(get_latest_tag "$dir")
  mod=$(get_module_path "$dir")

  local new_tag
  if [[ -z "$last_tag" ]]; then
    # No previous tag — start at v0.1.0.
    new_tag="${prefix}v0.1.0"
    echo "  ★ ${new_tag}  (first release — ${mod})"
  else
    # Extract version from tag and bump.
    local current_ver
    current_ver="${last_tag#${prefix}v}"
    local new_ver
    new_ver=$(bump_version "$current_ver" "$level")
    new_tag="${prefix}v${new_ver}"
    echo "  ★ ${new_tag}  (${last_tag} → ${new_tag})  — ${mod}"
  fi

  # Check if tag already exists.
  if git tag -l "$new_tag" | grep -q .; then
    echo "    ⚠ Tag ${new_tag} already exists, skipping."
    return 0
  fi

  # Create annotated tag.
  git tag -a "$new_tag" -m "Release ${new_tag} — ${mod}"
  echo "    ✓ Tag created."
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────

if [[ "$ALL" == "true" ]]; then
  echo "==> Scanning all modules for changes (level: ${LEVEL})..."
  echo ""

  # Find all go.mod directories, excluding example (not a library).
  count=0
  while IFS= read -r dir; do
    if [[ "$dir" == "example" ]]; then
      continue
    fi
    if has_changes "$dir"; then
      release_module "$dir" "$LEVEL"
      count=$((count + 1))
    fi
  done < <(find . -name go.mod -not -path './vendor/*' -print0 | xargs -0 -n1 dirname | sed 's|^\./||' | sort)

  echo ""
  if [[ $count -eq 0 ]]; then
    echo "No modules with changes found."
  else
    echo "Done. ${count} module(s) tagged."
    echo "Push with: make push-tags"
  fi
else
  # Single module release.
  if [[ -z "$PKG" ]]; then
    echo "Error: --pkg is required (or use --all)" >&2
    exit 1
  fi

  if [[ ! -d "$PKG" ]]; then
    echo "Error: directory not found: $PKG" >&2
    exit 1
  fi

  if [[ ! -f "${PKG}/go.mod" ]]; then
    echo "Error: no go.mod found in: $PKG" >&2
    exit 1
  fi

  echo "==> Releasing module: ${PKG} (level: ${LEVEL})"
  release_module "$PKG" "$LEVEL"
  echo ""
  echo "Push with: git push origin <tag-name>  or  make push-tags"
fi
