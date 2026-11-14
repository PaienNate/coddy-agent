#!/usr/bin/env bash
# Copies the `make android` binaries into the app's jniLibs under the names
# Android requires for an executable: lib<name>.so under lib/<abi>.
#
# Usage:
#   make android TAGS="http ui scheduler memory"
#   android/scripts/stage-binaries.sh
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
repo="$(cd "${root}/.." && pwd)"
build="${repo}/build"

stage() {
    local src="$1" abi="$2"
    if [ ! -f "$src" ]; then
        echo "missing $src - run 'make android TAGS=\"http ui scheduler memory\"' first" >&2
        exit 1
    fi
    local dest="${root}/app/src/main/jniLibs/${abi}"
    mkdir -p "$dest"
    install -m 0755 "$src" "${dest}/libcoddy.so"
    echo "staged ${src} -> ${dest}/libcoddy.so"
}

stage "${build}/coddy-android-arm64" arm64-v8a
stage "${build}/coddy-android-amd64" x86_64
