# devtool

`devtool` scaffolds a per-project `dev` build CLI plus a `Makefile`, and ships
shared helpers (`build/`, `packaging/`, `deploy/`, ...) for cross-compiling
Go applications - including cgo-heavy ones - to amd64, arm64 and armv7
Linux targets.

The cross-compile environment itself ships as a set of OCI images
("gobuild") published to **both** registries on every relevant push:

- **Forgejo** — [`.forgejo/workflows/build-images.yml`](.forgejo/workflows/build-images.yml)
- **GitHub** — [`.github/workflows/build-images.yml`](.github/workflows/build-images.yml) → `ghcr.io`

Local builds pull those images and bind-mount the source tree, so
contributors do not need to install cross toolchains on their host.

## gobuild build images

### Matrix

Each CI run (Forgejo or GitHub) produces 18 images:

| dimension       | values                          |
|-----------------|---------------------------------|
| Go minor        | `1.24`, `1.25` (floating patch) |
| Debian codename | `buster`, `bookworm`, `trixie`  |
| Flavor          | `base`, `wails`, `audio`        |

Flavors:

- `base` - lean cross-compile image (no desktop / audio extras).
- `wails` - adds GTK 3 + webkit2gtk for Wails / GTK UI apps, plus the
  `wails` CLI installed for the native amd64 toolchain.
- `audio` - adds `liblinphone-dev` + `libasound2-dev` for SIP /
  softphone / low-level audio cgo callers (amd64 native only - these
  packages are not reliably available for armhf / arm64 on Debian).

Two Dockerfiles back the matrix:

- [`docker/Dockerfile.gobuild`](docker/Dockerfile.gobuild) - bookworm and
  trixie. FROM the official `golang:<GO_VERSION>-<CODENAME>` image.
- [`docker/Dockerfile.gobuild-buster`](docker/Dockerfile.gobuild-buster) -
  legacy buster. FROM `debian:buster-slim`, pins apt at
  `archive.debian.org`, fetches Go from `go.dev` with SHA-256
  verification. **Treat buster images as legacy-only - no security
  updates flow through `archive.debian.org`.**

Both Dockerfiles share [`docker/scripts/`](docker/scripts/):

- `install-cross-toolchains.sh` - `crossbuild-essential-arm{hf,64}` plus
  multi-arch `libudev` / `libusb` so cgo can link against shared system
  libraries for all three target architectures.
- `install-wails-deps.sh` - GTK 3 / webkit2gtk. Only runs when
  `FLAVOR=wails`.
- `install-audio-deps.sh` - libasound2-dev / liblinphone-dev. Only runs
  when `FLAVOR=audio`. Keeping these separate from `wails` lets you
  pull a thin Wails image without dragging the linphone dependency
  chain in, and vice versa.
- `bootstrap-go-std.sh` - pre-warms `go install std` for amd64, arm64
  and armv7 using the canonical CGO flags below. This file is also
  installed into the image (at `/usr/local/lib/gobuild/`) so the
  `go-cross` helper can source it for its env vars.
- `go-cross` - convenience wrapper, installed to `/usr/local/bin/go-cross`.

### Canonical CGO flags

The cross-compile flags exist in exactly one place
([`docker/scripts/bootstrap-go-std.sh`](docker/scripts/bootstrap-go-std.sh)):

| Target | CC / CXX / AR prefix                | CGO_CFLAGS                                        |
|--------|-------------------------------------|---------------------------------------------------|
| armv7  | `arm-linux-gnueabihf-gcc` / `g++` / `ar` | `-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard` |
| arm64  | `aarch64-linux-gnu-gcc` / `g++` / `ar`   | `-march=armv8-a`                                  |
| amd64  | system gcc                          | none                                              |

[`build/golang.go`](build/golang.go) uses the same values when invoking
`go build` from the host (e.g. through the `dev` CLI). Mixing these
flags with anything else - in particular swapping `-march=armv7-a+fp`
for the explicit `-mfpu=vfpv3-d16 -mfloat-abi=hard` form - has caused
subtle ABI breakage in the past.

### Tag scheme

```
<registry>/<namespace>/gobuild:<go-minor>-<codename>[-<flavor>]
```

The flavor suffix is omitted for the default `base` flavor, otherwise
it is `-<flavor>` literally.

Examples:

| registry | example tag |
|----------|-------------|
| Forgejo (`forgejo.gophertribe.com/gophertribe`) | `forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm` |
| GHCR (`ghcr.io/<owner>`) | `ghcr.io/gophertribe/gobuild:1.25-bookworm` |

More tags: `1.24-trixie`, `1.25-buster-wails`, `1.25-bookworm-audio` (same
suffix rules on both registries).

Patch versions float: the workflow resolves the newest `go1.24.x` /
`go1.25.x` from `https://go.dev/dl/?mode=json` at job start, so the
published tag stays at the minor level while the underlying bits stay
fresh.

## CI configuration

Both workflows share the same matrix, tag scheme, floating Go patch
resolution, and smoke test. They differ only in registry authentication
and cache backend (Forgejo uses registry cache; GitHub uses GHA cache).

Trigger sources (each workflow watches its own file plus `docker/**`):
push to `main`, weekly cron (Monday 04:17 UTC), `workflow_dispatch`.

### Forgejo

Set on the Forgejo repository:

| kind     | name             | example value               | purpose                       |
|----------|------------------|-----------------------------|-------------------------------|
| variable | `REGISTRY`       | `forgejo.gophertribe.com`   | container registry hostname   |
| variable | `NAMESPACE`      | `gophertribe`               | namespace / package owner     |
| secret   | `FORGEJO_USER`   | `bot-publisher` (or actor)  | login for `docker login`      |
| secret   | `FORGEJO_TOKEN`  | (access token)              | needs `package:write` scope   |

Published as: `${REGISTRY}/${NAMESPACE}/gobuild:<tag>`

Requires a runner label `docker` (as configured in the workflow).

### GitHub Actions

Published as: `ghcr.io/${{ github.repository_owner }}/gobuild:<tag>`

No extra secrets are required: the workflow grants `packages: write` and
logs in to `ghcr.io` with `GITHUB_TOKEN`. After the first publish, make
the package public under **Settings → Packages → gobuild → Package
settings** if you want anonymous `docker pull`.

Optional repository variable:

| kind     | name               | purpose                                      |
|----------|--------------------|----------------------------------------------|
| variable | `GHCR_IMAGE_NAME`  | image name segment (default `gobuild`)       |

Uses `ubuntu-latest` runners with Docker Buildx.

The smoke step of every job runs

```sh
docker run --rm <image> sh -euc '
  go version
  go-cross amd64 go version
  go-cross arm64 go version
  go-cross armv7 go version
'
```

so a broken cross toolchain in the image fails the build before it is
tagged.

## Building locally

The exact command the CI runs:

```bash
docker buildx build \
  --platform linux/amd64 \
  --build-arg GO_VERSION=1.25.3 \
  --build-arg DEBIAN_CODENAME=bookworm \
  --build-arg FLAVOR=base \
  -f docker/Dockerfile.gobuild \
  -t gobuild:dev .
```

For the wails variant:

```bash
docker buildx build \
  --platform linux/amd64 \
  --build-arg GO_VERSION=1.25.3 \
  --build-arg DEBIAN_CODENAME=bookworm \
  --build-arg FLAVOR=wails \
  -f docker/Dockerfile.gobuild \
  -t gobuild:dev-wails .
```

For buster (note: must pass the **full** `M.m.p` version since there is
no upstream `golang:1.25-buster` image and the script needs an exact
tarball name):

```bash
docker buildx build \
  --platform linux/amd64 \
  --build-arg GO_VERSION=1.25.3 \
  --build-arg FLAVOR=base \
  -f docker/Dockerfile.gobuild-buster \
  -t gobuild:dev-buster .
```

## Using the image

```bash
# Forgejo registry
docker run --rm -it -v "$(pwd):/src" -w /src \
  forgejo.gophertribe.com/gophertribe/gobuild:1.25-bookworm bash

# GitHub Container Registry (same tag scheme)
docker run --rm -it -v "$(pwd):/src" -w /src \
  ghcr.io/gophertribe/gobuild:1.25-bookworm bash

# inside:
go-cross arm64 go build ./cmd/myapp
go-cross armv7 go test ./...
go-cross amd64 env | grep -E '^(GOARCH|CC|CGO_)'
```

### Overriding the image from the `dev` CLI

[`build/docker.go`](build/docker.go) composes the image reference from
defaults that can be overridden in three ways, in order of precedence:

1. `DockerBuildOpts.Image` - set programmatically by the calling CLI.
2. `DOCKER_BUILD_IMAGE` env var - full image reference, used verbatim.
3. `DockerBuildOpts.GoMinor` / `Codename` / `Flavor` fields - composed
   into the canonical tag via `BuildImageRef(...)`.

Point at either registry with the same tag:

```bash
# Forgejo (default composed ref in build/docker.go)
DOCKER_BUILD_IMAGE=forgejo.gophertribe.com/gophertribe/gobuild:1.24-bookworm dev build

# GHCR
DOCKER_BUILD_IMAGE=ghcr.io/gophertribe/gobuild:1.24-bookworm dev build
```

### Private modules

The image **does not** bake any credentials. At runtime,
[`build/docker.go`](build/docker.go) bind-mounts `~/.netrc` (or
`~/.gobuild_netrc` if present) into `/root/.netrc` inside the container,
which is what `go mod download` consults for `GOPRIVATE` hosts.

If you genuinely need a credential at *image build* time (e.g. private
module pulled by `go install` during Dockerfile build), pass it via a
BuildKit secret rather than a build-arg:

```bash
echo "machine github.com login x-access-token password $GH_TOKEN" > /tmp/netrc
docker buildx build \
  --secret id=netrc,src=/tmp/netrc \
  ... # then RUN --mount=type=secret,id=netrc,target=/root/.netrc go install ...
shred -u /tmp/netrc
```

## Scaffolding a new project

```bash
go install github.com/gophertribe/devtool/cmd/devtool@latest
cd path/to/your/repo
devtool init
```

This drops a `Makefile` and a minimal `cmd/dev` cobra app into the
repository, both wired to the helpers under [`build/`](build/).
