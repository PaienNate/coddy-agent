package llm

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRawCapturePassThroughWhenDisabled(t *testing.T) {
	// No recorder installed (the default): the transport must pass the request
	// straight through without reading or cloning the body.
	SetRawCaptureRecorder(nil)
	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", strings.NewReader("exact body"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "secret")
	response := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
	sentinelReq := req
	rt := &httpTraceTransport{inner: diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		if got != sentinelReq {
			t.Error("disabled capture cloned the request")
		}
		if got.Body != req.Body {
			t.Error("disabled capture replaced the body")
		}
		return response, nil
	})}
	got, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if got != response {
		t.Fatal("response changed")
	}
}

func TestRawCaptureRecordsAndRestores(t *testing.T) {
	ring := NewRawCaptureRecorder(10)

	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1/chat", strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "abc")

	// The inner transport sees the full body again, not the truncated capture.
	var innerBody string
	inner := diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(got.Body)
		innerBody = string(b)
		resp := &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("streamed answer")),
		}
		resp.Header = http.Header{}
		resp.Header.Set("Authorization", "resp-secret")
		return resp, nil
	})

	resp, err := captureExchange(ring, inner, req)
	if err != nil {
		t.Fatal(err)
	}
	if innerBody != `{"hello":"world"}` {
		t.Fatalf("inner body not restored, got %q", innerBody)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ReqBody != `{"hello":"world"}` {
		t.Errorf("req body = %q", e.ReqBody)
	}
	if e.RespBody != "streamed answer" {
		t.Errorf("resp body = %q", e.RespBody)
	}
	if e.Status != 201 {
		t.Errorf("status = %d", e.Status)
	}
	if _, ok := e.ReqHeader["Authorization"]; ok {
		t.Error("request Authorization header not stripped")
	}
	if _, ok := e.RespHeader["Authorization"]; ok {
		t.Error("response Authorization header not stripped")
	}
	if e.ReqHeader.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type lost: %#v", e.ReqHeader)
	}
	if e.ReqHeader.Get("X-Request-ID") != "abc" {
		t.Errorf("X-Request-ID lost: %#v", e.ReqHeader)
	}
}

func TestRawCaptureStripsVendorCredentialHeaders(t *testing.T) {
	ring := NewRawCaptureRecorder(10)

	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/messages?key=sk-url-secret&keep=1", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	// The credential headers the providers Coddy speaks actually set.
	req.Header.Set("Authorization", "Bearer sk-auth")
	req.Header.Set("x-api-key", "sk-anthropic")
	req.Header.Set("api-key", "sk-azure")
	req.Header.Set("x-goog-api-key", "sk-google")
	req.Header.Set("Proxy-Authorization", "Basic abc")
	req.Header.Set("Cookie", "sid=secret")
	req.Header.Set("Content-Type", "application/json")

	inner := diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		// The live request keeps its credentials; only the capture strips them.
		if got.Header.Get("x-api-key") != "sk-anthropic" {
			t.Error("live request lost its x-api-key")
		}
		resp := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
		resp.Header = http.Header{}
		resp.Header.Set("x-api-key", "sk-response")
		resp.Header.Set("Set-Cookie", "sid=secret")
		return resp, nil
	})
	resp, err := captureExchange(ring, inner, req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	for _, h := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Api-Key", "Api-Key", "X-Goog-Api-Key"} {
		if _, ok := e.ReqHeader[h]; ok {
			t.Errorf("request header %s leaked into the capture", h)
		}
	}
	if _, ok := e.RespHeader["X-Api-Key"]; ok {
		t.Error("response x-api-key leaked into the capture")
	}
	if _, ok := e.RespHeader["Set-Cookie"]; ok {
		t.Error("response Set-Cookie leaked into the capture")
	}
	if e.ReqHeader.Get("Content-Type") != "application/json" {
		t.Error("non-secret header was dropped")
	}
	// The URL query's credential is redacted; the rest survives.
	if strings.Contains(e.URL, "sk-url-secret") {
		t.Errorf("URL leaked the query key: %s", e.URL)
	}
	if !strings.Contains(e.URL, "keep=1") {
		t.Errorf("URL lost its non-secret query: %s", e.URL)
	}
	if !strings.Contains(e.URL, "REDACTED") {
		t.Errorf("URL did not mark the redaction: %s", e.URL)
	}
}

func TestRawCaptureBoundedBody(t *testing.T) {
	ring := NewRawCaptureRecorder(10)

	big := strings.Repeat("x", rawCaptureTextLimit+1000)
	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	inner := diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		// The inner transport still reads the whole body.
		b, _ := io.ReadAll(got.Body)
		if len(b) != len(big) {
			t.Errorf("inner body truncated to %d bytes", len(b))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	resp, err := captureExchange(ring, inner, req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if len(entries[0].ReqBody) != rawCaptureTextLimit {
		t.Errorf("captured req body length = %d, want %d", len(entries[0].ReqBody), rawCaptureTextLimit)
	}
}

func TestRawCaptureResponsePrefixOnly(t *testing.T) {
	ring := NewRawCaptureRecorder(10)

	req, _ := http.NewRequest(http.MethodGet, "http://unused.invalid/v1", nil)
	longResp := strings.Repeat("y", rawCaptureTextLimit+500)
	inner := diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(longResp))}, nil
	})
	resp, err := captureExchange(ring, inner, req)
	if err != nil {
		t.Fatal(err)
	}
	// The caller still reads the full response.
	full, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !reflect.DeepEqual(string(full), longResp) {
		t.Error("caller did not receive the full response body")
	}
	if len(ring.Entries()) != 1 || len(ring.Entries()[0].RespBody) != rawCaptureTextLimit {
		t.Errorf("captured resp body length = %d, want %d", len(ring.Entries()[0].RespBody), rawCaptureTextLimit)
	}
}

func TestRawCaptureRingClearsAndBounds(t *testing.T) {
	ring := NewRawCaptureRecorder(2)
	inner := diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("r"))}, nil
	})
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://unused.invalid/v1", nil)
		resp, err := captureExchange(ring, inner, req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	if got := len(ring.Entries()); got != 2 {
		t.Fatalf("ring cap not honored, count = %d", got)
	}
	ring.Clear()
	if got := len(ring.Entries()); got != 0 {
		t.Fatalf("clear left %d entries", got)
	}
}

// TestHTTPTraceTransportCapturesWhenRecorderInstalled proves the capture is
// wired into the transport the client builders actually return: installing the
// recorder makes a plain httpTraceTransport round trip record the exchange.
func TestHTTPTraceTransportCapturesWhenRecorderInstalled(t *testing.T) {
	ring := NewRawCaptureRecorder(10)
	SetRawCaptureRecorder(ring)
	defer SetRawCaptureRecorder(nil)

	response := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
	rt := &httpTraceTransport{inner: diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return response, nil
	})}
	req, _ := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", strings.NewReader(`{"a":1}`))
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := len(ring.Entries()); got != 1 {
		t.Fatalf("httpTraceTransport did not capture, count = %d", got)
	}
}

// readCountingReader counts the reads its consumer makes, so a test can prove
// a body was not drained behind the consumer's back.
type readCountingReader struct {
	inner io.Reader
	reads *int
}

func (r readCountingReader) Read(p []byte) (int, error) {
	*r.reads++
	return r.inner.Read(p)
}

// TestRawCaptureRequestBodyNotReadUpfront proves the capture tees the request
// body instead of draining it into memory: an inner transport that fails
// before sending anything leaves the body untouched (nothing was on the wire,
// so the entry keeps no prefix of it either). Reading the whole request body
// up front, as an earlier version did, would hold a body of any size in
// memory at once - tens of MiB for a prompt with images.
func TestRawCaptureRequestBodyNotReadUpfront(t *testing.T) {
	ring := NewRawCaptureRecorder(10)

	var reads int
	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", readCountingReader{
		inner: strings.NewReader("whole body"),
		reads: &reads,
	})
	if err != nil {
		t.Fatal(err)
	}
	inner := diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})
	if _, err := captureExchange(ring, inner, req); err == nil {
		t.Fatal("want the transport error to surface")
	}
	if reads != 0 {
		t.Fatalf("capture drained the request body up front, %d reads before anyone sent it", reads)
	}
	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].ReqBody != "" {
		t.Errorf("captured a request body nobody sent: %q", entries[0].ReqBody)
	}
	if entries[0].Err == "" {
		t.Error("transport error not recorded")
	}
	// Reading the body now - after the snapshot - proves the stream the
	// consumer would have sent is still whole; the tee records this read,
	// but the snapshot above was taken before it.
	rest, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(rest) != "whole body" {
		t.Fatalf("body after capture = %q, want the untouched original", rest)
	}
}

// TestUninstallRawCaptureRecorder covers the stop path's guard: only the
// recorder a surface itself installed comes out, so a stopping server never
// disturbs one another server installed in the meantime.
func TestUninstallRawCaptureRecorder(t *testing.T) {
	t.Cleanup(func() { SetRawCaptureRecorder(nil) })

	first := NewRawCaptureRecorder(2)
	second := NewRawCaptureRecorder(2)
	SetRawCaptureRecorder(first)
	if UninstallRawCaptureRecorder(second) {
		t.Error("uninstalled a recorder that was not installed")
	}
	if GetRawCaptureRecorder() != first {
		t.Fatal("the installed recorder was disturbed")
	}
	if !UninstallRawCaptureRecorder(first) {
		t.Error("the installed recorder did not come out")
	}
	if GetRawCaptureRecorder() != nil {
		t.Error("a recorder is still installed")
	}
	if UninstallRawCaptureRecorder(first) {
		t.Error("uninstall succeeded with nothing installed")
	}
	if UninstallRawCaptureRecorder(nil) {
		t.Error("uninstall of nil reported success")
	}
}
