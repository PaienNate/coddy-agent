package dev.coddy.app

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Intent
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.widget.Button
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import java.net.HttpURLConnection
import java.net.URL
import java.util.concurrent.atomic.AtomicBoolean

/**
 * The launcher: start and stop the server, open the page, and read its log.
 *
 * A WebView alone cannot say why a server did not come up - a phone shows a
 * blank page and an exit code nobody can read. This screen keeps the process
 * output on the device: the command it was started with, every line it
 * printed, and the code it exited with.
 */
class MainActivity : AppCompatActivity() {

    companion object {
        private const val POLL_MS = 600L
    }

    private lateinit var status: TextView
    private lateinit var logView: TextView
    private lateinit var logScroll: ScrollView
    private lateinit var startButton: Button
    private lateinit var stopButton: Button
    private lateinit var openButton: Button

    private val handler = Handler(Looper.getMainLooper())
    private var lastLogSize = 0
    private var probeInFlight = AtomicBoolean(false)
    private var probed = false

    private val tick = object : Runnable {
        override fun run() {
            render()
            handler.postDelayed(this, POLL_MS)
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        status = findViewById(R.id.coddy_status)
        logView = findViewById(R.id.coddy_log)
        logScroll = findViewById(R.id.coddy_log_scroll)
        startButton = findViewById(R.id.coddy_start)
        stopButton = findViewById(R.id.coddy_stop)
        openButton = findViewById(R.id.coddy_open)

        startButton.setOnClickListener { startServer() }
        stopButton.setOnClickListener { stopServer() }
        openButton.setOnClickListener { openCoddy() }
        findViewById<Button>(R.id.coddy_copy).setOnClickListener { copyLog() }
        findViewById<Button>(R.id.coddy_clear).setOnClickListener {
            CoddyState.clear()
            deleteLogFile()
            lastLogSize = 0
            render()
        }

        // A first launch starts the server, so opening the app is still enough;
        // after a crash the buttons take over and the log says why.
        if (CoddyState.phase == CoddyState.Phase.STOPPED) startServer()
        render()
    }

    override fun onResume() {
        super.onResume()
        handler.post(tick)
    }

    override fun onPause() {
        handler.removeCallbacks(tick)
        super.onPause()
    }

    private fun startServer() {
        probed = false
        ContextCompat.startForegroundService(this, Intent(this, CoddyService::class.java))
        render()
    }

    private fun stopServer() {
        CoddyState.log(this, "stop requested")
        val intent = Intent(this, CoddyService::class.java).setAction(CoddyService.ACTION_STOP)
        ContextCompat.startForegroundService(this, intent)
    }

    /** Opens the page only when the server answers; otherwise says so. */
    private fun openCoddy() {
        openButton.isEnabled = false
        Thread({
            val code = probe("/coddy/info")
            runOnUiThread {
                openButton.isEnabled = true
                if (code in 200..399) {
                    startActivity(Intent(this, CoddyWebActivity::class.java))
                } else {
                    CoddyState.log(this, "the page is not ready: /coddy/info answered $code")
                    render()
                }
            }
        }, "coddy-open").start()
    }

    private fun copyLog() {
        val text = CoddyState.snapshot().joinToString("\n")
        val clipboard = getSystemService(ClipboardManager::class.java)
        clipboard?.setPrimaryClip(ClipData.newPlainText("Coddy log", text))
        CoddyState.log(this, "log copied (${CoddyState.snapshot().size} lines)")
    }

    private fun render() {
        val lines = CoddyState.snapshot()
        status.text = when (CoddyState.phase) {
            CoddyState.Phase.STOPPED -> getString(R.string.status_stopped)
            CoddyState.Phase.STARTING -> getString(R.string.status_starting)
            CoddyState.Phase.RUNNING ->
                getString(R.string.status_running, CoddyRuntime.portFor(this))
            CoddyState.Phase.EXITED ->
                getString(R.string.status_exited, CoddyState.exitCode ?: -1)
        }
        startButton.isEnabled = CoddyState.phase != CoddyState.Phase.RUNNING &&
            CoddyState.phase != CoddyState.Phase.STARTING
        stopButton.isEnabled = CoddyState.phase == CoddyState.Phase.RUNNING ||
            CoddyState.phase == CoddyState.Phase.STARTING
        openButton.isEnabled = CoddyState.phase == CoddyState.Phase.RUNNING

        if (lines.size != lastLogSize) {
            lastLogSize = lines.size
            logView.text = lines.joinToString("\n")
            logScroll.post { logScroll.fullScroll(ScrollView.FOCUS_DOWN) }
        }
        // Once the process answers, ask the two routes the app depends on and
        // put the status codes in the log: a page that is not served says so
        // here instead of appearing as a black screen.
        if (CoddyState.phase == CoddyState.Phase.RUNNING && !probed &&
            probeInFlight.compareAndSet(false, true)
        ) {
            Thread({
                try {
                    val info = probe("/coddy/info")
                    val root = probe("/")
                    CoddyState.log(applicationContext, "probe /coddy/info -> $info, / -> $root")
                    if (root == 404) {
                        CoddyState.log(
                            applicationContext,
                            "the server answers 404 on /: this build carries no web UI " +
                                "(the ui build tag is missing)",
                        )
                    }
                    probed = true
                } finally {
                    probeInFlight.set(false)
                }
            }, "coddy-probe").start()
        }
    }

    private fun probe(path: String): Int {
        return try {
            val port = CoddyRuntime.portFor(this)
            val conn = URL("http://127.0.0.1:$port$path").openConnection() as HttpURLConnection
            conn.connectTimeout = 1500
            conn.readTimeout = 1500
            CoddyAuth.cookieHeader()?.let { conn.setRequestProperty("Cookie", it) }
            val code = conn.responseCode
            conn.disconnect()
            code
        } catch (e: Exception) {
            -1
        }
    }

    private fun deleteLogFile() {
        runCatching { java.io.File(filesDir, "coddy.log").delete() }
    }
}
