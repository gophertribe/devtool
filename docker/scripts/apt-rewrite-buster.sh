#!/usr/bin/env bash
# apt-rewrite-buster.sh
#
# Debian buster moved to archive.debian.org after the LTS window closed.
# This script rewrites the in-image apt sources to point at the archive,
# disables security/updates repositories (which no longer exist), and
# tells apt to accept the now-stale Valid-Until headers.
#
# Intended to be the first step in the buster Dockerfile, before any
# apt-get update is attempted.

set -euo pipefail

# buster-backports is intentionally omitted: on archive.debian.org it
# routinely wins multi-arch resolution and pulls newer libudev1 (e.g.
# 247.x from backports) that conflicts with libudev-dev from buster
# main (241-7~deb10u8), breaking crossbuild-essential / libc6-dev.
cat > /etc/apt/sources.list <<'EOF'
deb http://archive.debian.org/debian buster main contrib non-free
EOF

rm -f /etc/apt/sources.list.d/*.list

mkdir -p /etc/apt/apt.conf.d
cat > /etc/apt/apt.conf.d/99-archive <<'EOF'
Acquire::Check-Valid-Until "false";
APT::Get::AllowUnauthenticated "false";
APT::Install-Recommends "0";
APT::Install-Suggests "0";
APT::Default-Release "buster";
EOF

cat > /etc/dpkg/dpkg.cfg.d/01_nodoc <<'EOF'
path-exclude /usr/share/doc/*
path-include /usr/share/doc/*/copyright
path-exclude /usr/share/man/*
path-exclude /usr/share/groff/*
path-exclude /usr/share/info/*
path-exclude /usr/share/lintian/*
path-exclude /usr/share/linda/*
EOF
