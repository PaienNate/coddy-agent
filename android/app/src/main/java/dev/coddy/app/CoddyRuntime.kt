package dev.coddy.app

import android.content.Context
import android.util.Log
import java.io.File
import java.nio.file.Files
import java.nio.file.Paths

/**
 * What the app needs to start the server: where the binary and its busybox
 * live, where CODDY_HOME is, the environment, and the first-run config.
 *
 * The binary is packaged as a native library (`libcoddy.so`) because Android
 * 10 and later refuse to execute a file in the app's data directory, while a
 * file under nativeLibraryDir is executable. The same applies to busybox,
 * whose applets become symlinks in `<files>/bin` (busybox dispatches on
 * argv[0], and an APK cannot carry the symlinks itself).
 */
class CoddyRuntime(private val context: Context) {

    companion object {
        private const val TAG = "CoddyRuntime"

        /** Loopback port the server listens on. Fixed, so the WebView and the
         *  person's bookmarks agree; the app is the only client. */
        const val PORT = 8765

        private val APPLET_BLACKLIST = setOf(
            // Daemons and privileged helpers have no place in an app sandbox
            // and widen the surface an agent turn can reach.
            "httpd", "telnetd", "ftpd", "su", "login", "passwd",
        )
    }

    private val nativeDir = context.applicationInfo.nativeLibraryDir

    val binary = File(nativeDir, "libcoddy.so")
    val busybox = File(nativeDir, "libbusybox.so")

    val home = File(context.filesDir, "coddy").apply { mkdirs() }
    val workspace = File(context.filesDir, "workspace").apply { mkdirs() }
    private val tmp = File(context.cacheDir, "tmp").apply { mkdirs() }
    private val binDir = File(context.filesDir, "bin").apply { mkdirs() }
    private val config = File(home, "config.yaml")

    fun isReady(): Boolean {
        if (!binary.exists()) return false
        // The zip may not carry the executable bit through every installer; the
        // app owns the extracted file, so setting it costs nothing.
        if (!binary.canExecute()) binary.setExecutable(true, true)
        return binary.canExecute()
    }

    /**
     * Prepares the first run (config, busybox applets) and returns the process
     * builder for `coddy serve` with the environment the binary expects on a
     * phone: CODDY_HOME in the app's files, TMPDIR in its cache (Go's Android
     * default /data/local/tmp is not writable), and PATH over the applet farm
     * so `sed`, `awk`, `tar` and the rest are the pinned busybox.
     */
    fun prepare(): ProcessBuilder {
        writeConfigIfMissing()
        installApplets()

        val command = listOf(
            binary.absolutePath,
            "serve",
            "-H", "127.0.0.1",
            "-P", PORT.toString(),
            "-config", config.absolutePath,
            "-cwd", workspace.absolutePath,
        )
        val builder = ProcessBuilder(command)
        val env = builder.environment()
        env["HOME"] = context.filesDir.absolutePath
        env["CODDY_HOME"] = home.absolutePath
        env["TMPDIR"] = tmp.absolutePath
        env["PATH"] = listOf(binDir.absolutePath, "/system/bin", "/system/xbin")
            .joinToString(":")
        // The agent streams long turns; an idle read deadline must not cut one.
        env["CODDY_ANDROID"] = "1"
        builder.redirectErrorStream(true)
        return builder
    }

    /** aarch64 vs x86_64 matters only to the APK: each ABI carries its own
     *  busybox, so nothing has to be chosen here. */
    private fun writeConfigIfMissing() {
        if (config.exists()) return
        // The settings screen saves the rest through the API; what the app
        // pins here is the loopback bind and the "no credential on an app-local
        // server" decision. `allow_insecure` silences the warnings about a
        // server without one and turns off the open-server guard, which is
        // about foreign Host headers and cross-origin writes.
        val document = """
            # Written by the Coddy Android app on first launch. The settings
            # screen edits this file through the API.
            httpserver:
              enable: true
              host: 127.0.0.1
              port: $PORT
              allow_insecure: true
        """.trimIndent() + "\n"
        config.writeText(document)
        config.setReadable(true, true)
        config.setWritable(true, true)
    }

    /**
     * Creates one symlink per busybox applet in `<files>/bin`. Busybox decides
     * what to do from the name it was called by, so `bin/grep` is grep. An
     * applet that fails to link (already there, or an unsupported name) is
     * skipped: the farm is a convenience, not a dependency.
     */
    private fun installApplets() {
        if (!busybox.canExecute()) {
            Log.w(TAG, "busybox is missing; PATH keeps the system tools only")
            return
        }
        val listed = try {
            val process = ProcessBuilder(busybox.absolutePath, "--list")
                .redirectErrorStream(true)
                .start()
            process.inputStream.bufferedReader().readLines()
                .also { process.waitFor() }
        } catch (e: Exception) {
            Log.w(TAG, "cannot list busybox applets", e)
            return
        }
        var linked = 0
        for (name in listed) {
            val applet = name.trim()
            if (applet.isEmpty() || applet in APPLET_BLACKLIST) continue
            val link = File(binDir, applet)
            if (link.exists()) continue
            try {
                Files.createSymbolicLink(
                    Paths.get(link.absolutePath),
                    Paths.get(busybox.absolutePath),
                )
                linked++
            } catch (_: Exception) {
                // A name the filesystem refuses is not worth a crash.
            }
        }
        Log.i(TAG, "busybox: $linked applets linked in ${binDir.absolutePath}")
    }
}
