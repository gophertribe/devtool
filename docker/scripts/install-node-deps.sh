#!/usr/bin/env bash
# install-node-deps.sh
#
# Installs Node.js for CI / frontend tooling alongside the Go
# cross-compile toolchain. Only run when the Dockerfile is built with
# FLAVOR=node.
#
# NODE_MAJOR selects the major version stream (default 22).
#
# Strategy:
#   - bookworm / trixie: NodeSource apt packages (setup_${NODE_MAJOR}.x)
#   - buster (EOL): official linux-x64 tarball from nodejs.org with
#     SHA-256 verification. NodeSource has no buster packages.
#
# Debian apt nodejs packages are intentionally not used — they lag
# upstream and differ between releases.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

NODE_MAJOR="${NODE_MAJOR:-22}"

# shellcheck source=/dev/null
. /etc/os-release
codename="${VERSION_CODENAME:-}"

install_from_nodesource() {
  apt-get update
  apt-get install -y --no-install-recommends ca-certificates curl gnupg

  curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
  apt-get install -y --no-install-recommends nodejs

  apt-get clean
  rm -rf /var/lib/apt/lists/*
}

install_from_official_tarball() {
  # Official Node builds need glibc >= 2.28 (buster ships 2.28).
  local version version_num filename url expected_sha tmp

  echo "[install-node] resolving latest Node ${NODE_MAJOR}.x from nodejs.org"
  version="$(curl -fsSL https://nodejs.org/dist/index.json \
    | jq -r --arg m "${NODE_MAJOR}" '
        [.[] | select(.version | test("^v" + $m + "\\."))][0].version
      ')"
  if [ -z "${version}" ] || [ "${version}" = "null" ]; then
    echo "[install-node] could not resolve latest Node ${NODE_MAJOR}.x" >&2
    exit 1
  fi
  version_num="${version#v}"
  filename="node-${version}-linux-x64.tar.xz"
  url="https://nodejs.org/dist/${version}/${filename}"

  tmp="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -rf '${tmp}'" EXIT

  echo "[install-node] downloading ${url}"
  curl -fsSL --retry 3 --retry-delay 2 -o "${tmp}/${filename}" "${url}"

  echo "[install-node] verifying sha256"
  expected_sha="$(curl -fsSL "https://nodejs.org/dist/${version}/SHASUMS256.txt" \
    | awk -v f="${filename}" '$2 == f { print $1; exit }')"
  if [ -z "${expected_sha}" ]; then
    echo "[install-node] could not find sha256 for ${filename}" >&2
    exit 1
  fi
  echo "${expected_sha}  ${tmp}/${filename}" | sha256sum -c -

  echo "[install-node] extracting Node ${version_num} to /usr/local"
  tar -xJf "${tmp}/${filename}" -C /usr/local --strip-components=1 \
    --no-same-owner

  # Drop the EXIT trap so later cleanups in the Dockerfile are unaffected.
  trap - EXIT
  rm -rf "${tmp}"
}

if [ "${codename}" = "buster" ]; then
  install_from_official_tarball
else
  install_from_nodesource
fi

node --version
npm --version
