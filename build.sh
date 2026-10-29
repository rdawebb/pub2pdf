#!/bin/sh
# Build pub2pdf for this machine, or for another with an argument:
#   ./build.sh [mac|linux|windows]
# The program uses no cgo, so Go cross-compiles to any of them from any host
set -e
cd "$(dirname "$0")"

# Arch: the target's common one when cross-building, this machine's when
# native; $GOARCH overrides both
case "${1:-$(go env GOHOSTOS)}" in
    mac|darwin) os=darwin;  out=pub2pdf-mac;   arch=arm64 ;;
    linux)      os=linux;   out=pub2pdf-linux; arch=amd64 ;;
    windows)    os=windows; out=pub2pdf.exe;   arch=amd64 ;;
    *) echo "usage: $0 [mac|linux|windows]" >&2; exit 2 ;;
esac
if [ "$os" = "$(go env GOHOSTOS)" ]; then
    arch=$(go env GOHOSTARCH)
fi
arch=${GOARCH:-$arch}

if [ "$os" = windows ]; then
    # Embed version info, manifest and icon as a .syso, which the linker picks
    # up with no build flag; the _windows_<arch> suffix keeps it out of other
    # platforms' builds, which would reject it; only amd64 and arm64 handled
    ICON_FLAG=""
    [ -f icon.ico ] && ICON_FLAG="-icon=icon.ico"
    ARM_FLAG=""
    [ "$arch" = arm64 ] && ARM_FLAG="-arm"

    # shellcheck disable=SC2086 # deliberate word splitting: no icon means no flag
    go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 \
        -64 \
        $ARM_FLAG \
        -manifest=pub2pdf.manifest \
        $ICON_FLAG \
        -o "resource_windows_$arch.syso" \
        versioninfo.json
fi

# No -s -w: stripping saves a few hundred KB but makes the binary look packed,
# which antivirus heuristics flag, as they do a missing version resource
GOOS=$os GOARCH=$arch CGO_ENABLED=0 \
    go build -trimpath -o "$out" .

echo "Built $(pwd)/$out ($os/$arch)"
