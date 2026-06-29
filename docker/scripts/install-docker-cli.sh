#!/usr/bin/env bash
# install-docker-cli.sh
#
# Installs Docker CLI + buildx plugin (client only, no daemon). Only run
# when the Dockerfile is built with FLAVOR=node. CI jobs use the host
# daemon via /var/run/docker.sock (Forgejo runner: container.docker_host:
# automount).

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

apt-get install -y --no-install-recommends ca-certificates curl gnupg

install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/debian/gpg \
  | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

apt-get update
apt-get install -y --no-install-recommends docker-ce-cli docker-buildx-plugin

apt-get clean
rm -rf /var/lib/apt/lists/*
