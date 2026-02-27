#!/usr/bin/env bash
#
# Build script for sumicare-provider-bootc
# Clones bootc repo, syncs Cargo.toml versions, builds release binary
#

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BOOTC_DIR="${PROJECT_ROOT}/bootc-src"
BOOTC_VERSION="1.13.0"

sync_cargo_toml() {
    local src="${BOOTC_DIR}/Cargo.toml"
    local dst="${PROJECT_ROOT}/Cargo.toml"

    # Extract [workspace.dependencies] up to the next [workspace.metadata section
    local deps lints
    deps=$(sed -n '/^\[workspace\.dependencies\]/,/^\[workspace\.metadata/{ /^\[workspace\.metadata/d; p; }' "$src")
    # Extract [workspace.lints.*] sections to end of file
    lints=$(sed -n '/^\[workspace\.lints\./,$ p' "$src")

    cat > "$dst" << 'HEADER'
[workspace]
members = ["bootc-bridge", "bootc-src/crates/*"]
resolver = "2"

[profile.release]
lto = "thin"
panic = "abort"

[profile.thin]
inherits = "release"
debug = false
strip = true
lto = true
opt-level = "s"
codegen-units = 1

HEADER
    printf '%s\n\n%s\n' "$deps" "$lints" >> "$dst"
    echo "Synced Cargo.toml from bootc-src"
}

echo "=== Building sumicare-provider-bootc ==="

if [ ! -d "${BOOTC_DIR}/.git" ]; then
    echo "Cloning bootc repository..."
    git clone --depth 1 --branch "v${BOOTC_VERSION}" https://github.com/bootc-dev/bootc.git "${BOOTC_DIR}"
fi

sync_cargo_toml

echo "Building Rust staticlib..."
cd "${PROJECT_ROOT}"
cargo build --release -p bootc-bridge

echo "Building Go provider..."
go generate ./...
go build -ldflags="-X main.version=${BOOTC_VERSION}" -o terraform-provider-bootc .

echo "=== Build complete ==="
echo "Binary: ./terraform-provider-bootc"
