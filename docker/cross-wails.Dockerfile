FROM --platform=linux/amd64 gophertribe/gobuild:1.25-bookworm

RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

# ARMv7 (32-bit) with hardware floating point
RUN GOOS=linux GOARCH=arm GOARM=7 \
    CC=arm-linux-gnueabihf-gcc \
    CXX=arm-linux-gnueabihf-g++ \
    AR=arm-linux-gnueabihf-ar \
    CGO_CFLAGS="-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard" \
    CGO_CXXFLAGS="-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard" \
    go install github.com/wailsapp/wails/v2/cmd/wails@latest

# ARM64 (64-bit)
RUN GOOS=linux GOARCH=arm64 \
    CC=aarch64-linux-gnu-gcc \
    CXX=aarch64-linux-gnu-g++ \
    AR=aarch64-linux-gnu-ar \
    CGO_CFLAGS="-march=armv8-a" \
    CGO_CXXFLAGS="-march=armv8-a" \
    go install github.com/wailsapp/wails/v2/cmd/wails@latest