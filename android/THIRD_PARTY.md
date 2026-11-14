# Third-party binaries in the Android build

## BusyBox

The APK ships a statically linked BusyBox so an agent turn on the phone has
the POSIX tools Android does not carry: `sed`, `awk`, `grep`, `find`, `xargs`,
`tar`, `gzip`, `sort`, `uniq`, `tr`, `cut`, `diff`, `mktemp`, `timeout`, `nc`,
`base64`, `xxd`, `md5sum`, and BusyBox's `ash`.

- **Version:** 1.36.1-r31, from Alpine Linux's `busybox-static` package for
  `aarch64` and `x86_64`, branch v3.20.
- **Source:** https://dl-cdn.alpinelinux.org/alpine/v3.20/main/aarch64/busybox-static-1.36.1-r31.apk
  (and the `x86_64` sibling). `android/scripts/fetch-busybox.sh` pins each
  package by SHA-256 and refuses a mismatch.
- **Licence:** GNU General Public License, version 2. The full text ships in
  the app's APK asset `licenses/GPL-2.0.txt` and at
  https://www.gnu.org/licenses/old-licenses/gpl-2.0.html. The corresponding
  source is the BusyBox 1.36.1 release tarball, https://busybox.net/downloads/busybox-1.36.1.tar.bz2,
  built by Alpine's packaging; Alpine publishes its build recipes and patches
  at https://gitlab.alpinelinux.org/alpine/aports/-/tree/v3.20/main/busybox.

BusyBox runs as a separate executable, so it is mere aggregation with Coddy's
MIT-licensed binary; the two are not linked into one work.
