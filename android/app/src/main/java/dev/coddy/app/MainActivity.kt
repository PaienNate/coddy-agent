package dev.coddy.app

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import java.net.HttpURLConnection
import java.net.URL

/**
 * The app's one window: a WebView on the loopback server this process runs.
 *
 * It shows a small "starting" page while the service brings `coddy serve` up,
 * then loads the SPA from 127.0.0.1. The page is the same one the desktop
 * build serves - the server, the sessions and the providers are all on this
 * phone.
 */
class MainActivity : AppCompatActivity() {

    companion object {
        private const val TAG = "MainActivity"
        private const val STARTING_HTML = """
            <!doctype html><html><head><meta charset="utf-8">
            <meta name="viewport" content="width=device-width,initial-scale=1">
            <style>
              :root { color-scheme: dark; }
              body { margin: 0; height: 100vh; display: grid; place-items: center;
                     font: 15px/1.5 system-ui, sans-serif; background: #101014; color: #e6e6ea; }
              .box { text-align: center; max-width: 22em; padding: 1.5em; }
              .spin { width: 28px; height: 28px; margin: 0 auto 1em; border-radius: 50%;
                      border: 3px solid #333; border-top-color: #8ab4f8; animation: r 1s linear infinite; }
              @keyframes r { to { transform: rotate(360deg); } }
              .err { color: #ff8a80; white-space: pre-wrap; text-align: left; }
              button { margin-top: 1em; padding: .5em 1em; border-radius: 8px; border: 1px solid #444;
                       background: #1c1c22; color: inherit; font: inherit; }
            </style></head><body><div class="box">
            <div class="spin"></div><div id="state">Starting Coddy…</div>
            <div class="err" id="err"></div><button id="retry" hidden>Try again</button>
            <script>
              document.getElementById('retry').onclick = () => location.reload();
              window.coddyFailed = (msg) => {
                document.querySelector('.spin').style.display = 'none';
                document.getElementById('state').textContent = 'Coddy did not start.';
                document.getElementById('err').textContent = msg || '';
                document.getElementById('retry').hidden = false;
              };
            </script></div></body></html>
        """

        private fun baseUrl(): String = "http://127.0.0.1:${CoddyRuntime.PORT}"
    }

    private lateinit var webView: WebView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        askNotificationPermission()
        startServerService()

        webView = WebView(this).apply {
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            settings.databaseEnabled = true
            settings.cacheMode = WebSettings.LOAD_DEFAULT
            settings.mediaPlaybackRequiresUserGesture = false
            settings.mixedContentMode = WebSettings.MIXED_CONTENT_NEVER_ALLOW
            // Only the loopback page is ours; nothing may navigate away.
            webViewClient = object : WebViewClient() {
                override fun shouldOverrideUrlLoading(
                    view: WebView,
                    request: WebResourceRequest,
                ): Boolean {
                    val host = request.url.host ?: return true
                    return host != "127.0.0.1" && host != "localhost"
                }
            }
            setBackgroundColor(0xFF101014.toInt())
        }
        setContentView(webView)
        webView.loadDataWithBaseURL(
            "about:blank",
            STARTING_HTML,
            "text/html",
            "utf-8",
            null,
        )
        waitForServer(attemptsLeft = 100)
    }

    private fun startServerService() {
        val intent = Intent(this, CoddyService::class.java)
        ContextCompat.startForegroundService(this, intent)
    }

    /**
     * Polls /coddy/info until the server answers, then loads the SPA. The
     * service logs why a start failed; the page says so and offers a retry.
     */
    private fun waitForServer(attemptsLeft: Int) {
        Thread({
            var remaining = attemptsLeft
            var lastError: String? = null
            while (remaining-- > 0) {
                try {
                    val conn = URL("${baseUrl()}/coddy/info").openConnection() as HttpURLConnection
                    conn.connectTimeout = 1000
                    conn.readTimeout = 1000
                    val code = conn.responseCode
                    conn.disconnect()
                    if (code in 200..499) {
                        runOnUiThread { webView.loadUrl(baseUrl() + "/") }
                        return@Thread
                    }
                    lastError = "HTTP $code"
                } catch (e: Exception) {
                    lastError = e.message
                }
                try {
                    Thread.sleep(300)
                } catch (_: InterruptedException) {
                    return@Thread
                }
            }
            Log.e(TAG, "server did not answer: $lastError")
            val message = (lastError ?: "").replace("'", "\\'")
            runOnUiThread {
                webView.evaluateJavascript("window.coddyFailed('$message')", null)
            }
        }, "coddy-wait").start()
    }

    private fun askNotificationPermission() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED
        ) {
            return
        }
        ActivityCompat.requestPermissions(
            this,
            arrayOf(Manifest.permission.POST_NOTIFICATIONS),
            1,
        )
    }

    @Deprecated("The WebView owns history; the system back goes back a page.")
    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }
}
