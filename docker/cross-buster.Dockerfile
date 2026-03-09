FROM --platform=linux/amd64 buildpack-deps:buster-scm

# install cgo-related dependencies
RUN set -eux; \
	apt-get update; \
	apt-get install -y --no-install-recommends \
		g++ gcc libc6-dev make pkg-config libudev-dev libusb-1.0-0-dev \
		git-core fakeroot build-essential libasound2-dev liblinphone-dev libgtk-3-dev libwebkit2gtk-4.0-dev curl \
	; \
	rm -rf /var/lib/apt/lists/*

ENV PATH /usr/local/go/bin:$PATH

ENV GOLANG_VERSION 1.24.3

RUN set -eux; \
	arch="$(dpkg --print-architecture)"; arch="${arch##*-}"; \
	url=; \
	case "$arch" in \
		'amd64') \
			url='https://dl.google.com/go/go1.24.3.linux-amd64.tar.gz'; \
			;; \
		'armhf') \
			url='https://dl.google.com/go/go1.24.3.linux-armv6l.tar.gz'; \
			;; \
		'arm64') \
			url='https://dl.google.com/go/go1.24.3.linux-arm64.tar.gz'; \
			;; \
		'i386') \
			url='https://dl.google.com/go/go1.24.3.linux-386.tar.gz'; \
			;; \
		'mips64el') \
			url='https://dl.google.com/go/go1.24.3.linux-mips64le.tar.gz'; \
			;; \
		'ppc64el') \
			url='https://dl.google.com/go/go1.24.3.linux-ppc64le.tar.gz'; \
			;; \
		'riscv64') \
			url='https://dl.google.com/go/go1.24.3.linux-riscv64.tar.gz'; \
			;; \
		's390x') \
			url='https://dl.google.com/go/go1.24.3.linux-s390x.tar.gz'; \
			;; \
		*) echo >&2 "error: unsupported architecture '$arch' (likely packaging update needed)"; exit 1 ;; \
	esac; \
	\
	wget -O go.tgz "$url" --progress=dot:giga; \
	# Skip checksum and signature verification for Go 1.24.3 \
	tar -C /usr/local -xzf go.tgz; \
	rm go.tgz; \
	\
	go version

# don't auto-upgrade the gotoolchain
# https://github.com/docker-library/golang/issues/472
ENV GOTOOLCHAIN=local

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 1777 "$GOPATH"
WORKDIR $GOPATH

RUN printf '%s\n' 'path-exclude /usr/share/doc/*' 'path-include /usr/share/doc/*/copyright' 'path-exclude /usr/share/man/*' 'path-exclude /usr/share/groff/*' 'path-exclude /usr/share/info/*' 'path-exclude /usr/share/lintian/*' 'path-exclude /usr/share/linda/*' > /etc/dpkg/dpkg.cfg.d/01_nodoc && \
    echo 'APT::Install-Recommends "0" ; APT::Install-Suggests "0" ;' >> /etc/apt/apt.conf && export DEBIAN_FRONTEND=noninteractive && dpkg --add-architecture armhf && apt-get update && \
    apt-get install -yq fakeroot crossbuild-essential-armhf

ARG GITHUB_TOKEN
RUN echo "machine github.com login mklimuk password $GITHUB_TOKEN" > /root/.netrc

ENV PKG_CONFIG_PATH=/usr/lib/arm-linux-gnueabihf/pkgconfig

# bootstrap go environment for armhf
RUN CC=arm-linux-gnueabihf-gcc-8 GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 CGO_CFLAGS="-march=armv7-a+fp" CGO_CXXFLAGS="-march=armv7-a+fp" go install std
