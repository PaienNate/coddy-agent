package dev.coddy.app

/**
 * Masks credentials before a log line is stored, drawn, copied or written to
 * the log file.
 *
 * The launcher shows what the server prints, and a server's output can carry a
 * secret: a provider error that quotes a request URL, a config dump, an
 * `Authorization` header echoed back. The log also survives a crash on disk
 * and goes to the clipboard on one tap, so masking at the single choke point -
 * [CoddyState.log] - is what keeps a secret from outliving the screen it was
 * printed on.
 *
 * The patterns are deliberately narrow: a line that merely mentions a token
 * (for example "bearer auth" in the startup banner) stays readable, and only a
 * value that looks like a credential is replaced.
 */
object Redaction {

    private const val MASK = "***"

    // A credential header carries everything after the colon.
    private val credentialHeader = Regex(
        """(?i)((?:x-api-key|x-goog-api-key|authorization|proxy-authorization|cookie|set-cookie)\s*[:=]\s*)(.+)""",
    )
    private val bearer = Regex("""(?i)(bearer\s+)[A-Za-z0-9._~+/=\-]{8,}""")
    // The value stops at whitespace, a comma, a semicolon or an ampersand, so a
    // query string keeps the parameters after the secret one.
    private val secretAssignment = Regex(
        """(?i)((?:api[_-]?key|auth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|""" +
            """client[_-]?secret|secret|password|passwd|token)\s*[:=]\s*)(\"[^\"]*\"|'[^']*'|[^\s,;&]+)""",
    )
    private val apiKeyShape = Regex("""\bsk-[A-Za-z0-9_\-]{16,}""")
    private val secretFlag = Regex("""(--(?:auth-token|token|api-key|password)\s+)\S+""")

    /** One redacted line; idempotent, so a scrubbed file can be scrubbed again. */
    fun line(input: String): String {
        var text = input
        // A header name first: its value may itself say "Bearer ...", and the
        // whole value goes.
        text = credentialHeader.replace(text) { m -> m.groupValues[1] + MASK }
        text = bearer.replace(text) { m -> m.groupValues[1] + MASK }
        text = secretAssignment.replace(text) { m -> m.groupValues[1] + MASK }
        text = apiKeyShape.replace(text, "sk-$MASK")
        text = secretFlag.replace(text) { m -> m.groupValues[1] + MASK }
        return text
    }
}
