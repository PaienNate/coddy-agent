package llm

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RawCaptureEntry is one provider HTTP exchange the raw-capture recorder kept.
// It lives only in memory; nothing here is ever written to disk. The
// Authorization, Proxy-Authorization, Cookie and Set-Cookie headers are always
// stripped from both directions so a captured secret cannot sit in the ring.
type RawCaptureEntry struct {
	ID         string      `json:"id"`
	At         time.Time   `json:"at"`
	Method     string      `json:"method"`
	URL        string      `json:"url"`
	Status     int         `json:"status,omitempty"`
	ReqHeader  http.Header `json:"reqHeader,omitempty"`
	ReqBody    string      `json:"reqBody,omitempty"`
	RespHeader http.Header `json:"respHeader,omitempty"`
	RespBody   string      `json:"respBody,omitempty"`
	Err        string      `json:"err,omitempty"`

	// resp captures the response prefix lazily: the body is read by the caller
	// after RoundTrip returns, so the prefix is recorded as it is consumed and
	// read here at snapshot time. Not JSON-serialized.
	resp *captureReadCloser
}

// rawCaptureTextLimit bounds a captured request or response body, so a file or
// a huge stream cannot balloon the ring. The body a caller still needs is never
// truncated: only what the ring keeps is cut.
const rawCaptureTextLimit = 256 * 1024

// RawCaptureRecorder is a bounded, in-memory list of the last provider
// exchanges. It is a debug aid: the oldest entry is dropped once it fills.
type RawCaptureRecorder struct {
	mu      sync.Mutex
	entries []RawCaptureEntry
	cap     int
	seq     uint64
}

// NewRawCaptureRecorder returns a recorder holding at most cap entries.
func NewRawCaptureRecorder(cap int) *RawCaptureRecorder {
	if cap <= 0 {
		cap = 100
	}
	return &RawCaptureRecorder{cap: cap}
}

func (r *RawCaptureRecorder) add(e RawCaptureEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	e.ID = "raw_" + strconv.FormatUint(r.seq, 10)
	r.entries = append(r.entries, e)
	if len(r.entries) > r.cap {
		r.entries = r.entries[len(r.entries)-r.cap:]
	}
}

// Entries returns a copy of the captured exchanges, oldest first. A streamed
// response body that was still being read when the entry was made is now
// materialized, bounded to rawCaptureTextLimit, under the reader's lock.
func (r *RawCaptureRecorder) Entries() []RawCaptureEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RawCaptureEntry, len(r.entries))
	for i, e := range r.entries {
		if e.resp != nil {
			e.RespBody = e.resp.String()
		}
		e.resp = nil
		out[i] = e
	}
	return out
}

// Clear drops every captured exchange.
func (r *RawCaptureRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
}

// The recorder is process-wide and installed by the HTTP surface when
// httpserver.raw_capture is enabled. A nil recorder (the default) means the
// transport passes every request through untouched.
var (
	rawCaptureMu       sync.RWMutex
	rawCaptureRecorder *RawCaptureRecorder
)

// SetRawCaptureRecorder installs (non-nil) or removes (nil) the raw-capture
// recorder. The HTTP surface enables it when httpserver.raw_capture is on and
// clears it when the server stops.
func SetRawCaptureRecorder(r *RawCaptureRecorder) {
	rawCaptureMu.Lock()
	defer rawCaptureMu.Unlock()
	rawCaptureRecorder = r
}

// GetRawCaptureRecorder returns the installed recorder, or nil when capture is off.
func GetRawCaptureRecorder() *RawCaptureRecorder {
	rawCaptureMu.RLock()
	defer rawCaptureMu.RUnlock()
	return rawCaptureRecorder
}

// rawCaptureTransport wraps a round tripper and records the request and
// response bodies into the installed ring. When no recorder is installed it is
// a pass-through, so a process that ran without httpserver.raw_capture pays
// nothing but the one nil check.
type rawCaptureTransport struct {
	inner http.RoundTripper
}

// CloseIdleConnections forwards to the inner transport.
func (t *rawCaptureTransport) CloseIdleConnections() {
	if inner, ok := t.inner.(interface{ CloseIdleConnections() }); ok {
		inner.CloseIdleConnections()
	}
}

// RoundTrip captures one request/response pair before delegating to the inner
// transport.
func (t *rawCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ring := GetRawCaptureRecorder()
	if ring == nil {
		return t.inner.RoundTrip(req)
	}

	entry := RawCaptureEntry{
		At:        time.Now().UTC(),
		Method:    req.Method,
		URL:       redactedURL(req.URL),
		ReqHeader: sanitizedHeader(req.Header),
	}
	if req.Body != nil {
		// Read the whole request so the inner transport gets the body back; keep
		// only a bounded prefix in the entry.
		body, err := io.ReadAll(req.Body)
		if err == nil {
			entry.ReqBody = boundedText(body)
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		entry.Err = err.Error()
		ring.add(entry)
		return resp, err
	}
	entry.Status = resp.StatusCode
	entry.RespHeader = sanitizedHeader(resp.Header)
	if resp.Body != nil {
		// Wrap the live stream so the caller keeps it whole and the ring keeps
		// a bounded prefix, read lazily at snapshot time.
		reader := newCaptureReadCloser(resp.Body, rawCaptureTextLimit)
		resp.Body = reader
		entry.resp = reader
	}
	ring.add(entry)
	return resp, nil
}

// boundedText returns at most the first rawCaptureTextLimit bytes of b, so a
// large request body cannot balloon the ring.
func boundedText(b []byte) string {
	if len(b) > rawCaptureTextLimit {
		b = b[:rawCaptureTextLimit]
	}
	return string(b)
}

// sensitiveHeaders are the credential-carrying headers the capture must never
// keep. Besides the standard ones, the providers Coddy speaks set their key in
// a vendor header: Anthropic and its model list use x-api-key, some
// OpenAI-compatible gateways use api-key, and a Google-compatible endpoint uses
// x-goog-api-key. A captured credential would otherwise sit in the ring and be
// served by GET /coddy/raw-capture.
var sensitiveHeaders = map[string]struct{}{
	"Authorization":       {},
	"Proxy-Authorization": {},
	"Cookie":              {},
	"Set-Cookie":          {},
	"X-Api-Key":           {},
	"Api-Key":             {},
	"X-Goog-Api-Key":      {},
}

// sensitiveQueryParams are the query parameters that name a credential: a key
// some providers accept in the URL (`?key=`, `?api_key=`), or a signed token.
var sensitiveQueryParams = map[string]struct{}{
	"key":          {},
	"api_key":      {},
	"apikey":       {},
	"api-key":      {},
	"access_token": {},
	"token":        {},
}

// sanitizedHeader strips the credentials a captured headers map might carry.
func sanitizedHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	out := make(http.Header, len(h))
	for k, v := range h {
		if _, secret := sensitiveHeaders[http.CanonicalHeaderKey(k)]; secret {
			continue
		}
		out[k] = append([]string(nil), v...)
	}
	return out
}

// redactedURL drops the value of any credential query parameter so a key that
// travelled in the URL never reaches the ring.
func redactedURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.RawQuery == "" {
		return u.String()
	}
	q := u.Query()
	changed := false
	for name := range q {
		if _, secret := sensitiveQueryParams[strings.ToLower(name)]; secret {
			changed = true
			break
		}
	}
	if !changed {
		return u.String()
	}
	for name := range q {
		if _, secret := sensitiveQueryParams[strings.ToLower(name)]; secret {
			q.Set(name, "REDACTED")
		}
	}
	clone := *u
	clone.RawQuery = q.Encode()
	return clone.String()
}

// captureReadCloser passes every byte of the response through (so the caller
// sees the whole stream) while recording only the first limit bytes, guarded by
// a mutex because the caller reads the stream and a snapshot may run meanwhile.
type captureReadCloser struct {
	inner     io.ReadCloser
	mu        sync.Mutex
	buf       bytes.Buffer
	remaining int
}

func newCaptureReadCloser(body io.ReadCloser, limit int) *captureReadCloser {
	return &captureReadCloser{inner: body, remaining: limit}
}

func (c *captureReadCloser) Read(p []byte) (int, error) {
	n, err := c.inner.Read(p)
	if n > 0 {
		c.mu.Lock()
		if c.remaining > 0 {
			take := n
			if take > c.remaining {
				take = c.remaining
			}
			c.buf.Write(p[:take])
			c.remaining -= take
		}
		c.mu.Unlock()
	}
	return n, err
}

func (c *captureReadCloser) Close() error {
	return c.inner.Close()
}

// String returns the captured prefix, bounded already by the limit.
func (c *captureReadCloser) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}
