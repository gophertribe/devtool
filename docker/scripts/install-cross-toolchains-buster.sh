#!/usr/bin/env bash
# install-cross-toolchains-buster.sh
#
# Buster-specific variant of install-cross-toolchains.sh. On archive
# mirrors, adding armhf/arm64 before the native build-essential stack
# is in place causes apt to fail resolving libc6-dev / libudev-dev.
#
# docker.io/debian:buster-slim ships libc6 revisions (e.g. deb10u3)
# whose matching libc6-dev was never published on archive.debian.org.
# Do NOT pin libc6-dev to the hub image's libc6 version — sync both
# packages from archive with --allow-downgrades instead.
#
# Install order:
#   1. sync libc6 + libc6-dev from archive, then native toolchain
#   2. add foreign architectures
#   3. cross compilers and multi-arch -dev packages

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

echo "[install-cross-toolchains-buster] libc6 before sync: $(dpkg-query -W -f='${Version}' libc6 2>/dev/null || echo none)"
echo "[install-cross-toolchains-buster] libc6-dev candidates: $(apt-cache madison libc6-dev 2>/dev/null | awk '{print $3}' | paste -sd, - || echo none)"

# Let apt pick a consistent libc6 / libc6-dev pair from archive (usually
# downgrades runtime to match buster/main's libc6-dev). Fall back to the
# last known good pair on archive if the open-ended install fails.
if ! apt-get install -y --allow-downgrades --no-install-recommends \
  libc6 \
  libc6-dev; then
  echo "[install-cross-toolchains-buster] retrying with explicit buster/main glibc"
  apt-get install -y --allow-downgrades --no-install-recommends \
    libc6=2.28-10+deb10u1 \
    libc6-dev=2.28-10+deb10u1
fi

echo "[install-cross-toolchains-buster] libc6 after sync: $(dpkg-query -W -f='${Version}' libc6)"

apt-get install -y --no-install-recommends \
  bzip2 \
  make \
  patch \
  pkg-config \
  fakeroot \
  gcc \
  g++ \
  dpkg-dev \
  dpkg-cross \
  build-essential

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
