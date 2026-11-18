package dev.coddy.app

import android.content.Context
import android.webkit.CookieManager
import java.net.HttpURLConnection
import java.net.URL

/**
 * The app's own sign-in to its loopback server.
 *
 * The server binds 127.0.0.1, but a phone's loopback is shared with every app
 * on it: without a credential, any of them could call `GET /coddy/config` and
 * read the provider keys, or drive the agent. The built-in web sign-in closes
 * that: the config carries an argon2id hash, the app holds the matching
 * password in its private preferences, and every request travels with the
 * session cookie the login handed out.
 *
 * The password never reaches the page or the log; only the cookie does, and
 * the cookie is a server-side session that dies with the process.
 */
object CoddyAuth {

    private const val USER = "coddy"

    @Volatile
    private var cookie: String? = null

    /** The cookie to send with the launcher's own probes, when one is known. */
    fun cookieHeader(): String? = cookie

    fun clear() {
        cookie = null
    }

    fun user(): String = USER

    /**
     * Creates the sign-in account the first time it is missing: a random
     * password in the app's preferences, and its hash written into the config
     * by the binary's own `serve set-password`, so the hash format stays the
     * server's business.
     */
    fun ensureAccount(context: Context, runtime: CoddyRuntime) {
        val prefs = context.getSharedPreferences("coddy", Context.MODE_PRIVATE)
        val stored = prefs.getString("login_password", null)
        val hasHash = runCatching {
            runtime.config.exists() && runtime.config.readText().contains("password_hash:")
        }.getOrDefault(false)
        // A hash with no password (preferences lost, config restored) is
        // unusable: a new password replaces it. So is a password with no hash.
        val needsWrite = !hasHash || stored == null
        val password = stored ?: randomPassword().also {
            prefs.edit().putString("login_password", it).apply()
        }
        if (!needsWrite) return

        val result = runCatching { writeAccount(runtime, password) }
        if (result.isFailure) {
            CoddyState.log(context, "sign-in account was not written: ${result.exceptionOrNull()}")
        }
    }

    private fun writeAccount(runtime: CoddyRuntime, password: String): String {
        val process = ProcessBuilder(
            runtime.binary.absolutePath,
            "serve",
            "set-password",
            "-config", runtime.config.absolutePath,
            "-home", runtime.home.absolutePath,
            "-user", USER,
        )
            .redirectErrorStream(true)
            .apply {
                environment()["HOME"] = runtime.home.parentFile?.absolutePath ?: runtime.home.absolutePath
                environment()["CODDY_HOME"] = runtime.home.absolutePath
            }
            .start()
        process.outputStream.use { it.write((password + "\n").toByteArray()) }
        val output = process.inputStream.bufferedReader().readText().trim()
        val code = process.waitFor()
        if (code != 0) {
            error("set-password exited $code: $output")
        }
        return output
    }

    /**
     * Signs in and stores the cookie where both the WebView and the probes
     * find it. Runs after the server answers; a failure is logged and the page
     * will show the sign-in form instead.
     */
    fun signIn(context: Context, port: Int): Boolean {
        val prefs = context.getSharedPreferences("coddy", Context.MODE_PRIVATE)
        val password = prefs.getString("login_password", null) ?: return false
        val base = "http://127.0.0.1:$port"
        return try {
            val conn = URL("$base/coddy/auth/login").openConnection() as HttpURLConnection
            conn.requestMethod = "POST"
            conn.connectTimeout = 3000
            conn.readTimeout = 3000
            conn.doOutput = true
            conn.setRequestProperty("Content-Type", "application/json")
            // The route refuses a request it cannot place as same-origin, and a
            // native client sends no Sec-Fetch-Site; the Origin names this
            // server, which is where the request really comes from.
            conn.setRequestProperty("Origin", base)
            val body = """{"user":"$USER","password":"${jsonEscape(password)}"}"""
            conn.outputStream.use { it.write(body.toByteArray()) }
            val code = conn.responseCode
            if (code != HttpURLConnection.HTTP_OK) {
                CoddyState.log(context, "sign-in failed: HTTP $code")
                conn.disconnect()
                return false
            }
            val setCookie = conn.headerFields["Set-Cookie"]?.firstOrNull()
            conn.disconnect()
            val pair = setCookie?.substringBefore(";")?.trim().orEmpty()
            if (pair.isEmpty()) {
                CoddyState.log(context, "sign-in failed: no cookie in the answer")
                return false
            }
            cookie = pair
            val manager = CookieManager.getInstance()
            manager.setCookie("$base/", "$pair; Path=/")
            manager.flush()
            CoddyState.log(context, "signed in to the local server")
            true
        } catch (e: Exception) {
            CoddyState.log(context, "sign-in failed: $e")
            false
        }
    }

    private fun randomPassword(): String {
        val alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
        val random = java.security.SecureRandom()
        return (1..32).map { alphabet[random.nextInt(alphabet.length)] }.joinToString("")
    }

    private fun jsonEscape(value: String): String =
        value.replace("\\", "\\\\").replace("\"", "\\\"")
}
