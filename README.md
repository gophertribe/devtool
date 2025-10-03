# Docker build container setup

## Regular build

```bash
docker buildx build \
  --platform linux/amd64 \
-t gophertribe/gobuild:1.25-bookworm -f docker/cross-bookworm.Dockerfile .
```

## Building with secrets

```bash
# Create secret file
echo "your-github-token" > github_token.txt

# Build with secret
docker buildx build \
  --platform linux/amd64 \
  --secret id=github_token,src=github_token.txt \
  -t cross-compiler:latest \
  .

# Don't forget to remove the token file
rm github_token.txt
```

## Usage examples

```bash
# Run container
docker run -it -v $(pwd):/src gophertribe/gobuild:1.25-bookworm

# Inside container, use convenience scripts
go-arm go build ./...      # Builds for ARMv7
go-arm64 go build ./...    # Builds for ARM64

# Or set environment manually
export CC=arm-linux-gnueabihf-gcc
export GOARCH=arm
export GOOS=linux
go build ./...
```
