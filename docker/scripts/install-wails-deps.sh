#!/usr/bin/env bash
# install-wails-deps.sh
#
# Installs the desktop-side system libraries that Wails (v2) and other
# GTK/webkit based UIs need to compile against. Only run when the
# Dockerfile is built with FLAVOR=wails.
#
# Notes:
#   - Wails itself is a host-side build driver. We only need to install
#     these libraries for the native amd64 architecture; cross-compiling
#     a Wails app to ARM is typically done through Wails' own
#     cross-compile flow, which still relies on a native build host.
#   - Audio libraries (libasound, liblinphone) live in the separate
#     "audio" flavor (docker/scripts/install-audio-deps.sh). If you
#     need a Wails app that also embeds audio, build a composite image
#     downstream from gobuild:<...>-wails by layering the audio script.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

apt-get install -y --no-install-recommends \
  libgtk-3-dev \
  curl \
  git

if apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1; then
  apt-get install -y --no-install-recommends libwebkit2gtk-4.1-dev
else
  apt-get install -y --no-install-recommends libwebkit2gtk-4.0-dev
fi

apt-get clean
rm -rf /var/lib/apt/lists/*
