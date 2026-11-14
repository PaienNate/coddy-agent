#!/usr/bin/env bash
# Stages the Android build of Coddy and its busybox into the app's jniLibs.
#
# Android 10+ refuses to execute a file in the app's data directory, while a
# native library under lib/<abi> is executable - which is why both binaries are
# packaged as lib*.so. The APK cannot carry busybox's applet symlinks either;
# the app creates them in <files>/bin at first launch.
#
# The busybox is Alpine's busybox-static (musl, statically linked), pinned by
# version and SHA-256. BusyBox is GPLv2: see android/THIRD_PARTY.md for the
# licence and the source of the pinned build.
#
# Usage:
#   android/scripts/fetch-busybox.sh arm64
#   android/scripts/fetch-busybox.sh amd64
set -euo pipefail

arch="${1:-arm64}"
case "$arch" in
    arm64) pkg_arch=aarch64; abi=arm64-v8a ;;
    amd64) pkg_arch=x86_64;  abi=x86_64 ;;
    *) echo "usage: $0 arm64|amd64" >&2; exit 2 ;;
esac

# One release of Alpine's v3.20 branch, pinned. Bump both the version and the
# digest together; the digest is the .apk, not the unpacked binary.
version="1.36.1-r31"
url="https://dl-cdn.alpinelinux.org/alpine/v3.20/main/${pkg_arch}/busybox-static-${version}.apk"
case "$pkg_arch" in
    aarch64) sha256="1d8e7a7fc2ed69bdb8eb7be9d962489c21a0f75b72d8a325437c8a09d4cfac70" ;;
    x86_64)  sha256="1ca8a2894b073edea2a04231f335ad48795f92617c5eacae59458c51202068ed" ;;
esac

root="$(cd "$(dirname "$0")/.." && pwd)"
dest="${root}/app/src/main/jniLibs/${abi}"
mkdir -p "$dest"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

echo "fetching busybox ${version} (${pkg_arch})"
curl -fsSL --retry 3 -o "${work}/busybox.apk" "$url"
printf '%s  %s\n' "$sha256" "${work}/busybox.apk" | sha256sum -c -

# An .apk is a gzipped tar with a signature in front; the payload we want is
# the one static binary.
tar -xzf "${work}/busybox.apk" -C "$work" bin/busybox.static
install -m 0755 "${work}/bin/busybox.static" "${dest}/libbusybox.so"
echo "wrote ${dest}/libbusybox.so"
