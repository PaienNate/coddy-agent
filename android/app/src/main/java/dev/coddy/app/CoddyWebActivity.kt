package dev.coddy.app

import android.os.Bundle
import android.util.Log
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.appcompat.app.AppCompatActivity

/**
 * The page itself: one WebView on the loopback server this process runs. The
 * launcher owns start, stop and the log; this screen only draws the UI the
 * server serves.
 */
class CoddyWebActivity : AppCompatActivity() {

    companion object {
        private const val TAG = "CoddyWeb"
        private const val STARTING_HTML = """
            <!doctype html><html><head><meta charset="utf-8">
            <meta name="viewport" content="width=device-width,initial-scale=1">
            <style>
              :root { color-scheme: dark; }
              body { margin: 0; height: 100vh; display: grid; place-items: center;
                     font: 15px/1.5 system-ui, sans-serif; background: #101014; color: #e6e6ea; }
              .box { text-align: center; max-width: 24em; padding: 1.5em; }
              .spin { width: 28px; height: 28px; margin: 0 auto 1em; border-radius: 50%;
                      border: 3px solid #333; border-top-color: #8ab4f8; animation: r 1s linear infinite; }
              @keyframes r { to { transform: rotate(360deg); } }
              .err { color: #ff8a80; white-space: pre-wrap; text-align: left; margin-top: .6em; }
              button { margin-top: 1em; padding: .5em 1em; border-radius: 8px; border: 1px solid #444;
                       background: #1c1c22; color: inherit; font: inherit; }
            </style></head><body><div class="box">
            <div class="spin"></div><div id="state">Starting Coddy…</div>
            <div class="err" id="err"></div><button id="retry" hidden>Try again</button>
            <button id="back" hidden>Open the launcher log</button>
            <script>
              document.getElementById('retry').onclick = () => location.reload();
              document.getElementById('back').onclick = () => history.back();
              window.coddyFailed = (msg) => {
                document.querySelector('.spin').style.display = 'none';
                document.getElementById('state').textContent = 'Coddy did not start.';
                document.getElementById('err').textContent = msg || '';
                document.getElementById('retry').hidden = false;
                document.getElementById('back').hidden = false;
              };
            </script></div></body></html>
        """

        private fun baseUrl(context: android.content.Context): String =
            "http://127.0.0.1:${CoddyRuntime.portFor(context)}"
    }

    private lateinit var webView: WebView
    private lateinit var base: String

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        base = baseUrl(this)
        webView = WebView(this).apply {
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            settings.databaseEnabled = true
            settings.cacheMode = WebSettings.LOAD_DEFAULT
            settings.mediaPlaybackRequiresUserGesture = false
            settings.mixedContentMode = WebSettings.MIXED_CONTENT_NEVER_ALLOW
            // Only the loopback page is ours; nothing may navigate away.
            webViewClient = object : WebViewClient() {
                private var reauth = false

                override fun shouldOverrideUrlLoading(
                    view: WebView,
                    request: WebResourceRequest,
                ): Boolean {
                    val host = request.url.host ?: return true
                    return host != "127.0.0.1" && host != "localhost"
                }

                // A restarted server keeps no session, so the cookie the page
                // holds is dead: sign in again once and reload, rather than
                // leave a page that silently fails every request.
                override fun onReceivedHttpError(
                    view: WebView,
                    request: WebResourceRequest,
                    errorResponse: android.webkit.WebResourceResponse,
                ) {
                    if (errorResponse.statusCode != 401 || reauth) return
                    reauth = true
                    Thread({
                        if (CoddyAuth.signIn(this@CoddyWebActivity, CoddyRuntime.portFor(this@CoddyWebActivity))) {
                            runOnUiThread { view.reload() }
                        }
                        reauth = false
                    }, "coddy-reauth").start()
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
        if (CoddyAuth.cookieHeader() == null) {
            // The service normally signs in; a page opened before it finished,
            // or after a server restart, does it here.
            Thread({ CoddyAuth.signIn(this, CoddyRuntime.portFor(this)) }, "coddy-signin").start()
        }
        waitForServer(attemptsLeft = 100)
    }

    /**
     * Polls /coddy/info until the server answers, then loads the SPA. The
     * launcher's log holds the reason a start failed; this screen points back
     * to it instead of showing a black page.
     */
    private fun waitForServer(attemptsLeft: Int) {
        Thread({
            var remaining = attemptsLeft
            var lastError: String? = null
            while (remaining-- > 0) {
                try {
                    val conn = java.net.URL("$base/coddy/info")
                        .openConnection() as java.net.HttpURLConnection
                    conn.connectTimeout = 1000
                    conn.readTimeout = 1000
                    CoddyAuth.cookieHeader()?.let { conn.setRequestProperty("Cookie", it) }
                    val code = conn.responseCode
                    conn.disconnect()
                    if (code in 200..399) {
                        runOnUiThread { webView.loadUrl(base + "/") }
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

    @Deprecated("The WebView owns history; the system back goes back a page.")
    override fun onBackPressed() {
        if (webView.canGoBack()) {
            webView.goBack()
        } else {
            super.onBackPressed()
        }
    }
}
