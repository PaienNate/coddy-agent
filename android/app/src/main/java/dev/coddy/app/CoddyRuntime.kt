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

        /** The port the app asks for first; a busy one moves up the range. */
        const val DEFAULT_PORT = 8765
        private const val PORT_RANGE = 20
        private const val PREFS = "coddy"
        private const val PREF_PORT = "port"

        /** The loopback port the server uses, as the last start chose it. */
        fun portFor(context: Context): Int =
            context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getInt(PREF_PORT, DEFAULT_PORT)

        private fun canBind(port: Int): Boolean = try {
            java.net.ServerSocket(port, 1, java.net.InetAddress.getByName("127.0.0.1")).use { true }
        } catch (_: Exception) {
            false
        }

        /**
         * A phone is shared with every other app, and 127.0.0.1 is not
         * private: a port another process already holds makes `serve` exit 1
         * with "address already in use" and leaves the WebView talking to
         * whatever that process is - a bare "404 page not found" on a black
         * screen. The first free port in the range wins.
         */
        private fun choosePort(context: Context): Int {
            val wanted = portFor(context)
            for (port in wanted until wanted + PORT_RANGE) {
                if (canBind(port)) return port
            }
            return wanted
        }

        private val APPLET_BLACKLIST = setOf(
            // Daemons and privileged helpers have no place in an app sandbox
            // and widen the surface an agent turn can reach.
            "httpd", "telnetd", "ftpd", "su", "login", "passwd",
        )
    }

    private val nativeDir = context.applicationInfo.nativeLibraryDir

    val binary = File(nativeDir, "libcoddy.so")
    val busybox = File(nativeDir, "libbusybox.so")

    /** Resolved by [prepare]; the page and the probes read it back. */
    var port: Int = portFor(context)
        private set

    val home = File(context.filesDir, "coddy").apply { mkdirs() }
    val workspace = File(context.filesDir, "workspace").apply { mkdirs() }
    private val tmp = File(context.cacheDir, "tmp").apply { mkdirs() }
    private val binDir = File(context.filesDir, "bin").apply { mkdirs() }
    val config = File(home, "config.yaml")

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
        val wanted = portFor(context)
        port = choosePort(context)
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit().putInt(PREF_PORT, port).apply()
        if (port != wanted) {
            CoddyState.log(context, "port $wanted was busy; using $port")
        }
        writeConfigIfMissing()
        // The loopback port is shared with every other app: give the server
        // its own sign-in account before it starts, so the API is never
        // anonymous on the phone.
        CoddyAuth.ensureAccount(context, this)
        installApplets()
        CoddyState.log(context, "CODDY_HOME=${home.absolutePath}")
        CoddyState.log(context, "TMPDIR=${tmp.absolutePath}")
        CoddyState.log(context, "PATH=${binDir.absolutePath}:/system/bin:/system/xbin")

        val command = listOf(
            binary.absolutePath,
            "serve",
            "-H", "127.0.0.1",
            "-P", port.toString(),
            "-config", config.absolutePath,
            "-cwd", workspace.absolutePath,
        )
        // The path only. The file can carry a provider key or an API token,
        // and the launcher's log is drawn on screen, copied on one tap and
        // kept on disk.
        CoddyState.log(context, "config ${config.absolutePath} (contents not shown)")
        CoddyState.log(context, "exec ${command.joinToString(" ")}")
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
              port: $port
              allow_insecure: true
        """.trimIndent() + "\n"
        config.writeText(document)
        config.setReadable(true, true)
        config.setWritable(true, true)
        CoddyState.log(context, "wrote ${config.absolutePath}")
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
            CoddyState.log(context, "busybox: missing (system tools only)")
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
        CoddyState.log(context, "busybox: $linked applets linked")
    }
}
