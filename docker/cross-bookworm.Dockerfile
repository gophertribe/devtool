FROM --platform=linux/amd64 golang:1.25-bookworm

# Reduce image size by excluding documentation
RUN printf '%s\n' \
    'path-exclude /usr/share/doc/*' \
    'path-include /usr/share/doc/*/copyright' \
    'path-exclude /usr/share/man/*' \
    'path-exclude /usr/share/groff/*' \
    'path-exclude /usr/share/info/*' \
    'path-exclude /usr/share/lintian/*' \
    'path-exclude /usr/share/linda/*' \
    > /etc/dpkg/dpkg.cfg.d/01_nodoc && \
    echo 'APT::Install-Recommends "0" ; APT::Install-Suggests "0" ;' >> /etc/apt/apt.conf

# Install cross-compilation tools
RUN export DEBIAN_FRONTEND=noninteractive && \
    dpkg --add-architecture armhf && \
    dpkg --add-architecture arm64 && \
    apt-get update && \
    apt-get install -yq --no-install-recommends \
        fakeroot \
        crossbuild-essential-armhf \
        crossbuild-essential-arm64 \
        libudev-dev:armhf \
        libudev-dev:arm64 \
        libudev-dev:amd64 \
        ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set up pkg-config for cross-compilation
ENV PKG_CONFIG_PATH=/usr/lib/arm-linux-gnueabihf/pkgconfig:/usr/lib/aarch64-linux-gnu/pkgconfig

# Bootstrap Go standard library for target architectures
# Using more specific compiler versions and flags
ENV CGO_ENABLED=1

# ARMv7 (32-bit) with hardware floating point
RUN GOOS=linux GOARCH=arm GOARM=7 \
    CC=arm-linux-gnueabihf-gcc \
    CXX=arm-linux-gnueabihf-g++ \
    AR=arm-linux-gnueabihf-ar \
    CGO_CFLAGS="-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard" \
    CGO_CXXFLAGS="-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard" \
    go install -v std

# ARM64 (64-bit)
RUN GOOS=linux GOARCH=arm64 \
    CC=aarch64-linux-gnu-gcc \
    CXX=aarch64-linux-gnu-g++ \
    AR=aarch64-linux-gnu-ar \
    CGO_CFLAGS="-march=armv8-a" \
    CGO_CXXFLAGS="-march=armv8-a" \
    go install -v std

# AMD64 native compilation
RUN GOOS=linux GOARCH=amd64 go install -v std

# Embed GitHub token (disabled for now)
#RUN --mount=type=secret,id=github_token \
#     if [ -f /run/secrets/github_token ]; then \
#         mkdir -p /root/.netrc && \
#         echo "machine github.com login x-access-token password $(cat /run/secrets/github_token)" > /root/.netrc && \
#         chmod 600 /root/.netrc; \
#     fi

# Add convenience scripts for cross-compilation
RUN printf '#!/bin/bash\nexport CC=arm-linux-gnueabihf-gcc\nexport CXX=arm-linux-gnueabihf-g++\nexport AR=arm-linux-gnueabihf-ar\nexport GOOS=linux\nexport GOARCH=arm\nexport GOARM=7\nexport CGO_ENABLED=1\nexec "$@"\n' > /usr/local/bin/go-arm && \
    printf '#!/bin/bash\nexport CC=aarch64-linux-gnu-gcc\nexport CXX=aarch64-linux-gnu-g++\nexport AR=aarch64-linux-gnu-ar\nexport GOOS=linux\nexport GOARCH=arm64\nexport CGO_ENABLED=1\nexec "$@"\n' > /usr/local/bin/go-arm64 && \
    chmod +x /usr/local/bin/go-arm /usr/local/bin/go-arm64

# Labels for documentation
LABEL maintainer="michal@gophertribe.com" \
      description="Go cross-compilation environment for ARM architectures" \
      go.version="1.25" \
      architectures="amd64,armhf,arm64"