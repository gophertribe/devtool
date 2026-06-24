#!/usr/bin/env bash
# Applies a local patch to go-git until upstream merges
# https://github.com/go-git/go-git/pull/2171
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="v5.19.1"
MODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"
SRC="${MODCACHE}/github.com/go-git/go-git/v5@${VERSION}"
DST="${ROOT}/third_party/go-git"
PATCH="${ROOT}/patches/go-git-worktreeconfig.patch"
STAMP="${DST}/.patched"

if [[ ! -d "$SRC" ]]; then
	go mod download "github.com/go-git/go-git/v5@${VERSION}"
fi

if [[ -f "$STAMP" ]]; then
	exit 0
fi

rm -rf "$DST"
mkdir -p "$(dirname "$DST")"
cp -R "$SRC" "$DST"
chmod -R u+w "$DST"
patch -d "$DST" -p1 < "$PATCH"
touch "$STAMP"
