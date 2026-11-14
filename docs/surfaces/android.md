# Android app

The Android APK is the whole Coddy server on a phone: the same binary, the
same session store and the same web UI, running in the app's sandbox. The app
is not a shell that points at someone else's server - it starts **`coddy
serve`** on loopback and its WebView loads that page. The build lives in
**`android/`**, and the CI workflow **`.github/workflows/android-apk.yaml`**
produces a debug APK on every run.

## What it is

- **`libcoddy.so`** - the Go binary built by `make android`. Android 10 and
  later refuse to execute a file in the app's data directory, while a native
  library under `lib/<abi>` is executable, so the binary ships in `jniLibs`
  and `packaging.jniLibs.useLegacyPackaging = true` keeps it a real file.
- **`libbusybox.so`** - a pinned static BusyBox, see the next section.
- **`CoddyService`** - a foreground service running `libcoddy.so serve -H
  127.0.0.1 -P 8765`, so a turn keeps running while the person is in another
  app. Its notification is the price of that.
- **`MainActivity`** - one WebView on `http://127.0.0.1:8765/`, with a
  starting page that polls `/coddy/info` until the server answers.

Only loopback is exposed: providers, remote MCP servers and SSH hosts are
reached outbound, so the phone needs no inbound port and no LAN address.

## Build and install

CI builds the debug APK on every pull request that touches `android/` and on
`workflow_dispatch`; download **`coddy-android-apk`** from the run and install
it with `adb install coddy-android-<sha>-debug.apk`, or open the file on the
phone. A release build needs a signing key of your own.

Locally, with the Android SDK, an NDK, Go and Node installed:

```bash
android/scripts/build-apk.sh                       # arm64 and x86_64
ANDROID_ARCHS=arm64 android/scripts/build-apk.sh   # a phone only
```

The script runs `make android TAGS="http ui scheduler memory"`, stages the
binaries into `jniLibs`, fetches BusyBox and runs `gradle :app:assembleDebug`.

## First launch

The app writes **`CODDY_HOME/config.yaml`** with the loopback bind on the first
start. The web UI's onboarding then asks for a provider, exactly as on a
desktop server, and the settings screen edits the same file afterwards.

The state lives in the app's sandbox: `files/coddy` is `CODDY_HOME` (config,
sessions, SSH credentials), `files/workspace` is the default working
directory, and `cache/tmp` is `TMPDIR`, because Go's Android default
`/data/local/tmp` is not writable.

## SSH credentials

**Settings → SSH** stores the hosts **`ssh_run_command`** uses, in
**`<CODDY_HOME>/ssh-credentials.json`** with mode **0600**. A host carries a
password, a credential-helper command whose trimmed stdout is the password, a
private key path, or nothing - then the tool falls back to the SSH agent and
the keys under `~/.ssh` as it always did. Password authentication also answers
keyboard-interactive requests, which is how PAM-based servers ask for the same
secret.

The page never receives a password back: the list reports **has_password**,
and saving a host with an empty password keeps the stored one. **Test**
dials the host with the tool's authentication chain and closes without running
anything; a refusal comes back as the reason in words, and the page prints it.
Testing works for credentials not saved yet.

## BusyBox

Android's own tool set (toybox) varies by release and lacks most of what a
shell turn reaches for. The APK carries a statically linked BusyBox (Alpine's
`busybox-static`, pinned by version and SHA-256 in
**`android/scripts/fetch-busybox.sh`**) and, at first launch, links one applet
per name into `files/bin`, which the server's `PATH` puts first: `sed`, `awk`,
`grep`, `find`, `xargs`, `tar`, `gzip`, `sort`, `uniq`, `tr`, `cut`, `diff`,
`mktemp`, `timeout`, `nc`, `base64`, `xxd`, `md5sum` and BusyBox's `ash`.
Daemons such as `httpd` and `telnetd` are left unlinked on purpose.

BusyBox is GPLv2 and runs as a separate executable (mere aggregation with the
MIT-licensed binary). The licence text ships in the APK and the source is the
BusyBox 1.36.1 release, as **`android/THIRD_PARTY.md`** records.

The app is a BusyBox environment, not GNU: a command that needs `sed -i` with
GNU semantics or a tool the app does not bundle (`git`, `jq`, `python`,
`curl`) has to run on a remote host over SSH or through an MCP server. Web
pages are read with the built-in **`webfetch`** tool, not with a local
`curl`.

## Limits

- **No authentication on the loopback port.** Another app on the same phone
  could reach `127.0.0.1:8765`. The server is bound to loopback and the app is
  the only intended client; putting a token or a pairing step in front of it
  is the next hardening step, not something this build claims.
- **Fixed port 8765.** A second Coddy server on the phone waits for this one
  to stop.
- **No self-update.** `coddy update` replaces a binary on disk; here the
  binary is part of the APK, so an upgrade is a new APK.
- **The workspace is app-private.** Files outside it need another path in
  later work; the phone's shared storage is not mounted into a session.
