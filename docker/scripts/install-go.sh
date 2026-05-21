#!/usr/bin/env bash
# install-go.sh
#
# Downloads, verifies and installs an upstream Go toolchain into /usr/local.
# Only used by the buster Dockerfile, where the official golang:X-buster
# images do not exist for current Go releases.
#
# Inputs (env vars):
#   GO_VERSION  required, full Go version without the leading "go" prefix
#               (e.g. "1.25.3"). The matching tarball is fetched from
#               https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz and
#               its SHA-256 is cross-checked against the catalog at
#               https://go.dev/dl/?mode=json.
#
# Requires: curl, jq, tar (apt-installed before this script runs).

set -euo pipefail

: "${GO_VERSION:?GO_VERSION is required, e.g. 1.25.3}"

arch="amd64"
goos="linux"
filename="go${GO_VERSION}.${goos}-${arch}.tar.gz"
url="https://go.dev/dl/${filename}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "[install-go] resolving sha256 for go${GO_VERSION} ${goos}/${arch}"
expected_sha="$(curl -fsSL 'https://go.dev/dl/?mode=json&include=all' \
  | jq -r --arg v "go${GO_VERSION}" --arg os "$goos" --arg arch "$arch" '
      .[] | select(.version == $v) | .files[]
      | select(.os == $os and .arch == $arch and .kind == "archive")
      | .sha256')"

if [ -z "${expected_sha}" ] || [ "${expected_sha}" = "null" ]; then
  echo "[install-go] could not resolve sha256 for go${GO_VERSION} ${goos}/${arch} from go.dev catalog" >&2
  exit 1
fi

echo "[install-go] downloading ${url}"
curl -fsSL --retry 3 --retry-delay 2 -o "${tmp}/${filename}" "${url}"

echo "[install-go] verifying sha256"
echo "${expected_sha}  ${tmp}/${filename}" | sha256sum -c -

echo "[install-go] extracting to /usr/local"
rm -rf /usr/local/go
tar -C /usr/local -xzf "${tmp}/${filename}"

/usr/local/go/bin/go version
