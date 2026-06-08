#!/usr/bin/env bash
# Cut a release: build CHANGELOG section from conventional-commit log since the
# previous tag, open $EDITOR for review (skip with --yes), commit, tag, push.
# GitHub workflow `release.yml` picks up the tag and runs goreleaser.
set -euo pipefail

usage() {
  cat <<EOF
usage: $0 <version> [--yes]

  version   semver without leading v (e.g. 1.0.33)
  --yes     skip $EDITOR review, use auto-generated CHANGELOG section as-is

env:
  EDITOR    editor to use for review (default: vi)

guards:
  - working tree must be clean
  - must be on main
  - tag must not already exist
  - CHANGELOG section must contain at least one bullet
EOF
  exit "${1:-1}"
}

[[ $# -lt 1 ]] && usage
[[ "${1:-}" == "-h" || "${1:-}" == "--help" ]] && usage 0

V="${1#v}"
YES="${2:-}"

if ! [[ "$V" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]]; then
  echo "ERROR: version '$V' is not semver (expected N.N.N[-suffix])" >&2
  exit 1
fi

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "ERROR: working tree dirty. commit or stash first." >&2
  exit 1
fi

BRANCH=$(git branch --show-current)
if [[ "$BRANCH" != "main" ]]; then
  echo "ERROR: not on main (on $BRANCH). cut releases from main." >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/v${V}" >/dev/null; then
  echo "ERROR: tag v${V} already exists." >&2
  exit 1
fi

echo "==> fetching latest from origin"
git fetch --tags origin
git pull --ff-only origin main

PREV=$(git describe --tags --abbrev=0 2>/dev/null || true)
RANGE=""
if [[ -n "$PREV" ]]; then
  RANGE="${PREV}..HEAD"
  echo "==> collecting commits since $PREV"
else
  echo "==> no previous tag; collecting all commits"
fi

DATE=$(date +%Y-%m-%d)
TMP=$(mktemp -t release-changelog.XXXXXX)
trap 'rm -f "$TMP"' EXIT

{
  echo "## [v${V}] - ${DATE}"
  echo
  for type in feat fix perf refactor docs test chore; do
    matches=$(git log ${RANGE} --no-merges --pretty=format:"%s" 2>/dev/null \
      | grep -E "^${type}(\([^)]+\))?: " || true)
    if [[ -n "$matches" ]]; then
      label=$(printf '%s' "$type" | tr '[:lower:]' '[:upper:]' | cut -c1)$(printf '%s' "$type" | cut -c2-)
      echo "### ${label}"
      printf '%s\n' "$matches" | sed -E "s/^${type}(\([^)]+\))?: /- /"
      echo
    fi
  done
} > "$TMP"

if [[ "$YES" != "--yes" ]]; then
  echo "==> opening $TMP in ${EDITOR:-vi} for review"
  "${EDITOR:-vi}" "$TMP"
fi

if ! grep -qE "^- " "$TMP"; then
  echo "ERROR: empty CHANGELOG section (no '- ' bullets). aborting." >&2
  exit 1
fi

if [[ ! -f CHANGELOG.md ]]; then
  cat > CHANGELOG.md <<EOF
# Changelog

Notable changes per release. Newest first.
EOF
fi

echo "==> inserting section into CHANGELOG.md"
python3 - "$TMP" <<'PY'
import pathlib, sys
section = pathlib.Path(sys.argv[1]).read_text().rstrip() + "\n\n"
p = pathlib.Path("CHANGELOG.md")
old = p.read_text() if p.exists() else "# Changelog\n\n"
lines = old.splitlines(keepends=True)
out, inserted = [], False
for line in lines:
    if not inserted and line.startswith("## "):
        out.append(section)
        inserted = True
    out.append(line)
if not inserted:
    if out and not out[-1].endswith("\n"):
        out.append("\n")
    out.append("\n" + section)
p.write_text("".join(out))
PY

echo "==> committing and tagging v${V}"
git add CHANGELOG.md
git commit -m "chore: release v${V}"
git tag "v${V}"

echo "==> pushing main + tag"
git push origin main "v${V}"

cat <<EOF

==> v${V} pushed. release.yml triggered.
    monitor:  gh run watch -R bearded-giant/redis-tui
    release:  gh release view v${V} -R bearded-giant/redis-tui --web
EOF
