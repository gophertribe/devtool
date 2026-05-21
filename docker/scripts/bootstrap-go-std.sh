#!/usr/bin/env bash
# bootstrap-go-std.sh
#
# Single source of truth for the CGO cross-compile environment.
#
# Two roles:
#   1. When *executed*, it pre-warms (`go install std`) the Go standard
#      library for amd64, arm64 and armv7 using the canonical CGO flags,
#      so subsequent project builds inside the container reuse a hot
#      cache.
#   2. When *sourced* (by `/usr/local/bin/go-cross`, or interactively),
#      it just exports the env vars that describe each target's
#      toolchain prefix and CGO flags.
#
# All cross-compile flags MUST be defined here exactly once and consumed
# by both this script and the runtime helper. Drift between the image
# bootstrap flags and the per-project build flags causes subtle ABI
# breakage (notably with -mfloat-abi=hard on ARMv7).

set -eu

# --- Canonical target descriptors -------------------------------------

# ARMv7 (32-bit, hard float, vfpv3-d16) - matches Raspberry Pi 2/3/4 and
# most ARMv7 SBCs / embedded boards.
export ARMV7_CC=arm-linux-gnueabihf-gcc
export ARMV7_CXX=arm-linux-gnueabihf-g++
export ARMV7_AR=arm-linux-gnueabihf-ar
export ARMV7_CFLAGS="-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard"

# ARM64 (ARMv8 generic). No board-specific tuning - any -mcpu= override
# should be project-side.
export ARM64_CC=aarch64-linux-gnu-gcc
export ARM64_CXX=aarch64-linux-gnu-g++
export ARM64_AR=aarch64-linux-gnu-ar
export ARM64_CFLAGS="-march=armv8-a"

# pkg-config paths so cgo can resolve cross-arch libraries.
export PKG_CONFIG_PATH="${PKG_CONFIG_PATH:-/usr/lib/arm-linux-gnueabihf/pkgconfig:/usr/lib/aarch64-linux-gnu/pkgconfig}"

# --- Detect sourced vs executed --------------------------------------
# When sourced, ${BASH_SOURCE[0]} differs from $0. When executed, they
# match (or BASH_SOURCE is unset under sh; the fallback handles that).

_sourced=0
if [ -n "${BASH_SOURCE+x}" ]; then
  [ "${BASH_SOURCE[0]}" != "$0" ] && _sourced=1
fi

if [ "$_sourced" = "1" ]; then
  return 0 2>/dev/null || true
fi

# --- Bootstrap path (executed) ---------------------------------------

if ! command -v go >/dev/null 2>&1; then
  echo "[bootstrap-go-std] go binary not on PATH" >&2
  exit 1
fi

echo "[bootstrap-go-std] amd64"
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go install -v std

echo "[bootstrap-go-std] arm64"
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 \
  CC="$ARM64_CC" CXX="$ARM64_CXX" AR="$ARM64_AR" \
  CGO_CFLAGS="$ARM64_CFLAGS" CGO_CXXFLAGS="$ARM64_CFLAGS" \
  go install -v std

echo "[bootstrap-go-std] armv7"
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 \
  CC="$ARMV7_CC" CXX="$ARMV7_CXX" AR="$ARMV7_AR" \
  CGO_CFLAGS="$ARMV7_CFLAGS" CGO_CXXFLAGS="$ARMV7_CFLAGS" \
  go install -v std
