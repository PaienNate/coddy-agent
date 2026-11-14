# Coddy for Android (APK)

The whole server, on the phone. This directory is the Gradle project that
packages the Go binary and a static BusyBox into an APK whose WebView loads the
embedded SPA from `127.0.0.1` - nothing here talks to a remote Coddy.

- `scripts/build-apk.sh` - the whole build: `make android`, jniLibs staging,
  busybox, `gradle :app:assembleDebug`.
- `scripts/fetch-busybox.sh` - the pinned BusyBox (see `THIRD_PARTY.md`).
- `scripts/stage-binaries.sh` - copies `build/coddy-android-*` into
  `app/src/main/jniLibs/<abi>/libcoddy.so`.
- CI: `.github/workflows/android-apk.yaml` builds the debug APK and uploads it
  as an artifact.

Why the binaries are `lib*.so`: Android 10 and later refuse to `exec` a file
from the app's data directory, while a native library under `lib/<abi>` is
executable. `packaging.jniLibs.useLegacyPackaging = true` keeps them as real
files in the installed app.

The first launch writes `CODDY_HOME/config.yaml` with the loopback bind and
starts `coddy serve` as a foreground service; the SPA's onboarding then asks
for a provider. SSH credentials and their Test button live on the **Settings →
SSH** page.

Guide: `docs/surfaces/android.md`.
