#!/usr/bin/env bash
# install-cross-toolchains.sh
#
# Installs the GCC/G++ cross toolchains and the multi-arch system
# libraries we need to cgo-cross-compile to armhf (ARMv7) and arm64
# from an amd64 host image. Common to both the modern and the buster
# Dockerfile.
#
# Expects apt to already be functional (i.e. on buster the
# apt-rewrite-buster.sh step has already run and `apt-get update` has
# been issued by the caller).

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

dpkg --add-architecture armhf
dpkg --add-architecture arm64

apt-get update

apt-get install -y --no-install-recommends \
  build-essential \
  pkg-config \
  fakeroot \
  ca-certificates \
  crossbuild-essential-armhf \
  crossbuild-essential-arm64 \
  libudev-dev:amd64 \
  libudev-dev:armhf \
  libudev-dev:arm64 \
  libusb-1.0-0-dev:amd64 \
  libusb-1.0-0-dev:armhf \
  libusb-1.0-0-dev:arm64

apt-get clean
rm -rf /var/lib/apt/lists/*
