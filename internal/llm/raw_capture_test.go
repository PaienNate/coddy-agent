package llm

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRawCapturePassThroughWhenDisabled(t *testing.T) {
	// No recorder installed (the default): the transport must pass the request
	// straight through without reading or cloning the body.
	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", strings.NewReader("exact body"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "secret")
	response := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}
	sentinelReq := req
	rt := &rawCaptureTransport{inner: diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
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
	SetRawCaptureRecorder(ring)
	defer SetRawCaptureRecorder(nil)

	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1/chat", strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "abc")

	// The inner transport sees the full body again, not the truncated capture.
	var innerBody string
	rt := &rawCaptureTransport{inner: diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(got.Body)
		innerBody = string(b)
		resp := &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("streamed answer")),
		}
		resp.Header = http.Header{}
		resp.Header.Set("Authorization", "resp-secret")
		return resp, nil
	})}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if innerBody != `{"hello":"world"}` {
		t.Fatalf("inner body not restored, got %q", innerBody)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

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

func TestRawCaptureBoundedBody(t *testing.T) {
	ring := NewRawCaptureRecorder(10)
	SetRawCaptureRecorder(ring)
	defer SetRawCaptureRecorder(nil)

	big := strings.Repeat("x", rawCaptureTextLimit+1000)
	req, err := http.NewRequest(http.MethodPost, "http://unused.invalid/v1", strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	rt := &rawCaptureTransport{inner: diagnosticRoundTripFunc(func(got *http.Request) (*http.Response, error) {
		// The inner transport still reads the whole body.
		b, _ := io.ReadAll(got.Body)
		if len(b) != len(big) {
			t.Errorf("inner body truncated to %d bytes", len(b))
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	entries := ring.Entries()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if len(entries[0].ReqBody) != rawCaptureTextLimit {
		t.Errorf("captured req body length = %d, want %d", len(entries[0].ReqBody), rawCaptureTextLimit)
	}
	if _, ok := ring.Entries()[0].ReqHeader["Authorization"]; ok {
		t.Error("Authorization header leaked into captured entry")
	}
}

func TestRawCaptureResponsePrefixOnly(t *testing.T) {
	ring := NewRawCaptureRecorder(10)
	SetRawCaptureRecorder(ring)
	defer SetRawCaptureRecorder(nil)

	req, _ := http.NewRequest(http.MethodGet, "http://unused.invalid/v1", nil)
	longResp := strings.Repeat("y", rawCaptureTextLimit+500)
	rt := &rawCaptureTransport{inner: diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(longResp))}, nil
	})}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	// The caller still reads the full response.
	full, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !reflect.DeepEqual(string(full), longResp) {
		t.Error("caller did not receive the full response body")
	}
	if len(ring.Entries()) != 1 || len(ring.Entries()[0].RespBody) != rawCaptureTextLimit {
		t.Errorf("captured resp body length = %d, want %d", len(ring.Entries()[0].RespBody), rawCaptureTextLimit)
	}
}

func TestRawCaptureRingClearsAndBounds(t *testing.T) {
	ring := NewRawCaptureRecorder(2)
	SetRawCaptureRecorder(ring)
	defer SetRawCaptureRecorder(nil)
	rt := &rawCaptureTransport{inner: diagnosticRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("r"))}, nil
	})}
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://unused.invalid/v1", nil)
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if got := len(ring.Entries()); got != 2 {
		t.Fatalf("ring cap not honored, count = %d", got)
	}
	ring.Clear()
	if got := len(ring.Entries()); got != 0 {
		t.Fatalf("clear left %d entries", got)
	}
}
