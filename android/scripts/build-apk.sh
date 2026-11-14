#!/usr/bin/env bash
# Builds a debug APK locally: the Go server, busybox, the jniLibs and Gradle.
# The CI job (.github/workflows/android-apk.yaml) runs the same steps.
#
# Needs: Go, Node/npm (for the SPA), the Android SDK with NDK and platform 35,
# and Gradle 8.9 on PATH (or run this inside GitHub Actions, which provides
# all of them).
#
# Usage:
#   android/scripts/build-apk.sh
#   ANDROID_ARCHS="arm64" android/scripts/build-apk.sh   # one ABI, for a phone
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
repo="$(cd "${root}/.." && pwd)"

archs="${ANDROID_ARCHS:-arm64 amd64}"

echo "==> Go server for Android (${archs})"
make -C "$repo" android TAGS="http ui scheduler memory" ANDROID_ARCHS="$archs"

echo "==> staging the binaries"
"${root}/scripts/stage-binaries.sh"

echo "==> busybox"
for arch in $archs; do
    "${root}/scripts/fetch-busybox.sh" "$arch"
done

echo "==> APK"
cd "$root"
gradle :app:assembleDebug --no-daemon
echo "APK: ${root}/app/build/outputs/apk/debug/app-debug.apk"
