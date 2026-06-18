#!/usr/bin/env bash
# install-node-deps.sh
#
# Installs Node.js from NodeSource for CI / frontend tooling alongside the
# Go cross-compile toolchain. Only run when the Dockerfile is built with
# FLAVOR=node.
#
# NODE_MAJOR selects the NodeSource stream (e.g. 22 -> setup_22.x). Debian
# apt nodejs packages are intentionally not used — they lag upstream and
# differ between releases.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

NODE_MAJOR="${NODE_MAJOR:-22}"

apt-get update

apt-get install -y --no-install-recommends ca-certificates curl gnupg

curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
apt-get install -y --no-install-recommends nodejs

apt-get clean
rm -rf /var/lib/apt/lists/*
