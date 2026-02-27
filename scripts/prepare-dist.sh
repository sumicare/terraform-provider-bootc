#!/usr/bin/env bash
# Create zip archives from prebuilt binaries for GoReleaser release
set -euo pipefail

VERSION="${GORELEASER_VERSION:-${GITHUB_REF_NAME:-}}"
VERSION="${VERSION#v}"

if [ -z "$VERSION" ]; then
    echo "Error: Could not determine version"
    exit 1
fi

PROJECT="terraform-provider-bootc"
BUILD_NAME="sumicare-provider-bootc"
mkdir -p build

for arch in amd64 arm64; do
    src="prebuilt/linux_${arch}/${BUILD_NAME}_v${VERSION}"
    zip_name="${PROJECT}_${VERSION}_linux_${arch}.zip"

    if [ ! -f "$src" ]; then
        echo "Error: ${src} not found" >&2
        exit 1
    fi

    cp "$src" "${PROJECT}_v${VERSION}"
    zip "build/${zip_name}" "${PROJECT}_v${VERSION}"
    rm -f "${PROJECT}_v${VERSION}"
    echo "Created build/${zip_name}"
done
