package dev.coddy.app

import android.content.Context
import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * What the launcher draws and the log it shows: the server's own output, the
 * exact command it was started with, and the exit code. Everything here is
 * in-process state plus a file under the app's files, so an unexpected exit
 * leaves the last lines readable after the process is gone - which is what
 * makes "exit 1" diagnosable on a phone instead of a black screen.
 */
object CoddyState {

    enum class Phase { STOPPED, STARTING, RUNNING, EXITED }

    private const val MAX_LINES = 800
    private const val LOG_FILE = "coddy.log"
    private const val LOG_LIMIT_BYTES = 512 * 1024L

    @Volatile
    var phase: Phase = Phase.STOPPED
        private set

    @Volatile
    var exitCode: Int? = null
        private set

    private val lines = ArrayDeque<String>()
    private val listeners = mutableListOf<() -> Unit>()

    fun setStarting() {
        exitCode = null
        phase = Phase.STARTING
        notifyListeners()
    }

    fun setRunning() {
        phase = Phase.RUNNING
        notifyListeners()
    }

    fun setExited(code: Int) {
        exitCode = code
        phase = Phase.EXITED
        notifyListeners()
    }

    fun setStopped() {
        phase = Phase.STOPPED
        notifyListeners()
    }

    /** Appends one line, redacted; the file keeps what a crash takes with it. */
    fun log(context: Context?, line: String) {
        val stamped = "${timestamp()} ${Redaction.line(line)}"
        synchronized(lines) {
            lines.addLast(stamped)
            while (lines.size > MAX_LINES) lines.removeFirst()
        }
        if (context != null) {
            runCatching {
                val file = File(context.filesDir, LOG_FILE)
                if (file.exists() && file.length() > LOG_LIMIT_BYTES) file.delete()
                file.appendText(stamped + "\n")
            }
        }
        notifyListeners()
    }

    fun snapshot(): List<String> = synchronized(lines) { lines.toList() }

    /**
     * Rewrites a log file an earlier build wrote before it redacted: that build
     * dumped the whole config, which can carry a provider key or an API token.
     * Runs once at service start.
     */
    fun scrubLogFile(context: Context) {
        val file = File(context.filesDir, LOG_FILE)
        if (!file.exists()) return
        runCatching {
            val scrubbed = file.readLines().joinToString("\n") { Redaction.line(it) }
            file.writeText(scrubbed + "\n")
        }
    }

    fun clear() {
        synchronized(lines) { lines.clear() }
        notifyListeners()
    }

    fun addListener(listener: () -> Unit) {
        synchronized(listeners) { listeners.add(listener) }
    }

    fun removeListener(listener: () -> Unit) {
        synchronized(listeners) { listeners.remove(listener) }
    }

    private fun notifyListeners() {
        val copy = synchronized(listeners) { listeners.toList() }
        copy.forEach { runCatching { it() } }
    }

    private fun timestamp(): String =
        SimpleDateFormat("HH:mm:ss", Locale.US).format(Date())
}
