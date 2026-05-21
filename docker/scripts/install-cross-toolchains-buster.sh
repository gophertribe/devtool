#!/usr/bin/env bash
# install-cross-toolchains-buster.sh
#
# Buster-specific variant of install-cross-toolchains.sh. On archive
# mirrors, adding armhf/arm64 before the native build-essential stack
# is in place causes apt to fail resolving libc6-dev / libudev-dev.
# Hub images also ship a newer libc6 (deb10u3) than buster/main's
# libc6-dev (deb10u1) unless debian-security is enabled — see
# apt-rewrite-buster.sh.
#
# Install order:
#   1. native amd64 toolchain, libc6-dev matched to installed libc6
#   2. add foreign architectures
#   3. cross compilers and multi-arch -dev packages

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

libc6_ver="$(dpkg-query -W -f='${Version}' libc6 2>/dev/null || true)"
echo "[install-cross-toolchains-buster] libc6=${libc6_ver:-unknown}"

# bzip2 is required by dpkg-dev; make/patch by build-essential.
# Install libc6-dev at the same version as the base image's libc6
# before pulling in build-essential (avoids u1 vs u3 skew on archive).
base_pkgs=(
  bzip2
  make
  patch
  pkg-config
  fakeroot
  gcc
  g++
  dpkg-dev
  dpkg-cross
)

if [ -n "${libc6_ver}" ]; then
  apt-get install -y --no-install-recommends \
    "libc6-dev=${libc6_ver}" \
    "${base_pkgs[@]}"
else
  apt-get install -y --no-install-recommends \
    libc6-dev \
    "${base_pkgs[@]}"
fi

apt-get install -y --no-install-recommends build-essential

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
