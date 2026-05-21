#!/usr/bin/env bash
# install-cross-toolchains-buster.sh
#
# Buster-specific variant of install-cross-toolchains.sh. On archive
# mirrors, adding armhf/arm64 before the native build-essential stack
# is in place causes apt to fail resolving libc6-dev / libudev-dev
# (especially when backports are enabled). We therefore:
#   1. install native amd64 toolchain + dpkg-cross first
#   2. add foreign architectures
#   3. install cross compilers and multi-arch -dev packages

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

apt-get install -y --no-install-recommends \
  build-essential \
  pkg-config \
  fakeroot \
  dpkg-dev \
  dpkg-cross \
  gcc \
  g++ \
  libc6-dev

dpkg --add-architecture armhf
dpkg --add-architecture arm64

apt-get update

apt-get install -y --no-install-recommends \
  crossbuild-essential-armhf \
  crossbuild-essential-arm64 \
  libudev-dev \
  libudev-dev:armhf \
  libudev-dev:arm64 \
  libusb-1.0-0-dev \
  libusb-1.0-0-dev:armhf \
  libusb-1.0-0-dev:arm64

apt-get clean
rm -rf /var/lib/apt/lists/*
