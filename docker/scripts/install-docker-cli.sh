#!/usr/bin/env bash
# install-docker-cli.sh
#
# Installs Docker CLI + buildx plugin (client only, no daemon). Only run
# when the Dockerfile is built with FLAVOR=node. CI jobs use the host
# daemon via /var/run/docker.sock (Forgejo runner: container.docker_host:
# automount).
#
# Strategy:
#   - bookworm / trixie: Docker apt repository (docker-ce-cli +
#     docker-buildx-plugin)
#   - buster (EOL): official static CLI tarball from download.docker.com
#     plus the buildx plugin binary from GitHub releases. Docker's apt
#     repo no longer publishes buster packages.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

# shellcheck source=/dev/null
. /etc/os-release
codename="${VERSION_CODENAME:-}"

install_from_apt() {
  apt-get update
  apt-get install -y --no-install-recommends ca-certificates curl gnupg

  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian ${codename} stable" \
    > /etc/apt/sources.list.d/docker.list

  apt-get update
  apt-get install -y --no-install-recommends docker-ce-cli docker-buildx-plugin

  apt-get clean
  rm -rf /var/lib/apt/lists/*
}

install_from_static() {
  # Optional pins; empty means "resolve latest" at build time.
  local docker_version="${DOCKER_VERSION:-}"
  local buildx_version="${BUILDX_VERSION:-}"
  local tmp filename url plugin_dir

  tmp="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap "rm -rf '${tmp}'" EXIT

  if [ -z "${docker_version}" ]; then
    echo "[install-docker] resolving latest static Docker CLI"
    docker_version="$(curl -fsSL https://download.docker.com/linux/static/stable/x86_64/ \
      | grep -oE 'docker-[0-9]+\.[0-9]+\.[0-9]+\.tgz' \
      | sed 's/^docker-//;s/\.tgz$//' \
      | sort -V \
      | tail -1)"
  fi
  if [ -z "${docker_version}" ]; then
    echo "[install-docker] could not resolve static Docker CLI version" >&2
    exit 1
  fi

  filename="docker-${docker_version}.tgz"
  url="https://download.docker.com/linux/static/stable/x86_64/${filename}"
  echo "[install-docker] downloading ${url}"
  curl -fsSL --retry 3 --retry-delay 2 -o "${tmp}/${filename}" "${url}"

  echo "[install-docker] installing docker CLI ${docker_version} to /usr/local/bin"
  tar -xzf "${tmp}/${filename}" -C "${tmp}"
  install -m 0755 "${tmp}/docker/docker" /usr/local/bin/docker

  if [ -z "${buildx_version}" ]; then
    echo "[install-docker] resolving latest buildx release"
    buildx_version="$(curl -fsSL https://api.github.com/repos/docker/buildx/releases/latest \
      | jq -r '.tag_name' \
      | sed 's/^v//')"
  fi
  if [ -z "${buildx_version}" ] || [ "${buildx_version}" = "null" ]; then
    echo "[install-docker] could not resolve buildx version" >&2
    exit 1
  fi

  plugin_dir=/usr/local/lib/docker/cli-plugins
  install -d "${plugin_dir}"
  echo "[install-docker] downloading buildx v${buildx_version}"
  curl -fsSL --retry 3 --retry-delay 2 \
    -o "${plugin_dir}/docker-buildx" \
    "https://github.com/docker/buildx/releases/download/v${buildx_version}/buildx-v${buildx_version}.linux-amd64"
  chmod 0755 "${plugin_dir}/docker-buildx"

  trap - EXIT
  rm -rf "${tmp}"
}

if [ "${codename}" = "buster" ]; then
  install_from_static
else
  install_from_apt
fi

docker --version
docker buildx version
