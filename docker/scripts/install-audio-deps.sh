#!/usr/bin/env bash
# install-audio-deps.sh
#
# Installs the audio-related system libraries that cgo callers
# (SIP softphones, low-level audio capture/playback, etc.) typically
# link against. Only run when the Dockerfile is built with
# FLAVOR=audio.
#
# Native amd64 only - the cross-arch variants of liblinphone are not
# reliably packaged for armhf / arm64 on Debian. Projects that need
# cross-arch audio have to vendor / build linphone themselves.

set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update

apt-get install -y --no-install-recommends \
  libasound2-dev \
  liblinphone-dev

apt-get clean
rm -rf /var/lib/apt/lists/*
