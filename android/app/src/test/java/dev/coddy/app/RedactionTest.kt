package dev.coddy.app

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * The launcher log is drawn, copied and kept on disk; a credential that
 * reaches it must not survive [Redaction.line].
 */
class RedactionTest {

    @Test
    fun `an authorization header is masked whole`() {
        assertEquals(
            "Authorization: ***",
            Redaction.line("Authorization: Bearer abcdef1234567890"),
        )
        assertEquals("x-api-key: ***", Redaction.line("x-api-key: sk-live-1234567890"))
    }

    @Test
    fun `a secret assignment is masked`() {
        assertEquals("api_key: ***", Redaction.line("api_key: sk-abcdefghijklmnop1234"))
        assertEquals("auth_token=***", Redaction.line("auth_token=abc123def456"))
        assertEquals("password: ***", Redaction.line("password: \"hunter two\""))
        assertEquals("token = ***", Redaction.line("token = ghp_abcdefghijkl"))
    }

    @Test
    fun `a query parameter with a credential is masked`() {
        assertEquals(
            "https://api.example/v1?token=***&keep=1",
            Redaction.line("https://api.example/v1?token=deadbeef&keep=1"),
        )
        assertEquals(
            "https://api.example/v1?api_key=***",
            Redaction.line("https://api.example/v1?api_key=sk-abc"),
        )
    }

    @Test
    fun `a secret shaped value and a secret flag are masked`() {
        assertEquals("using sk-*** now", Redaction.line("using sk-abcDEF1234567890xyz now"))
        assertEquals("--auth-token ***", Redaction.line("--auth-token abc123"))
    }

    @Test
    fun `ordinary lines stay readable`() {
        val line = "listening addr=127.0.0.1:8765 auth=false"
        assertEquals(line, Redaction.line(line))
        assertEquals("httpserver http://127.0.0.1:8765 (bearer auth)", Redaction.line("httpserver http://127.0.0.1:8765 (bearer auth)"))
        assertEquals("token refresh finished", Redaction.line("token refresh finished"))
    }

    @Test
    fun `redaction is idempotent`() {
        val once = Redaction.line("api_key: sk-abcdefghijklmnop1234")
        assertEquals(once, Redaction.line(once))
    }
}
