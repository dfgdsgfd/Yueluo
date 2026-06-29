package services

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHiddenWatermarkRemoteClientEmbedAndExtract(t *testing.T) {
	var embedCalls int32
	payload := []byte("payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Internal-API-Key"); got != "secret" {
			t.Fatalf("X-Internal-API-Key = %q, want secret", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/v1/watermark/embed":
			atomic.AddInt32(&embedCalls, 1)
			if got := r.FormValue("payload_b64"); got != base64.StdEncoding.EncodeToString(payload) {
				t.Fatalf("payload_b64 = %q", got)
			}
			assertRemoteWatermarkFormOptions(t, r, HiddenWatermarkRemoteOptions{
				PasswordWM:  11,
				PasswordImg: 12,
			})
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("marked"))
		case "/v1/watermark/extract":
			if got := r.FormValue("payload_bytes_candidates"); got != "7" {
				t.Fatalf("payload_bytes_candidates = %q", got)
			}
			assertRemoteWatermarkFormOptions(t, r, HiddenWatermarkRemoteOptions{
				PasswordWM:  11,
				PasswordImg: 12,
			})
			file, _, err := r.FormFile("reference_file")
			if err != nil {
				t.Fatalf("reference_file: %v", err)
			}
			reference, _ := io.ReadAll(file)
			_ = file.Close()
			if string(reference) != "reference" {
				t.Fatalf("reference_file = %q", reference)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload_b64":   base64.StdEncoding.EncodeToString(payload),
				"payload_bytes": len(payload),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{
		URL:     server.URL + "/",
		APIKey:  "secret",
		Timeout: time.Second,
	})
	options := HiddenWatermarkRemoteOptions{
		PasswordWM:              11,
		PasswordImg:             12,
		OperationTimeoutSeconds: 45,
		Engine:                  "dwt_dct_svd",
		DWTDctSVDRepeat:         9,
		DWTDctSVDScale:          64,
		WatermarkWidth:          720,
		WatermarkHeight:         720,
		RecoverDimensions:       [][2]int{{720, 720}, {1280, 1280}},
	}
	marked, err := client.Embed(context.Background(), []byte("image"), payload, options)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if string(marked) != "marked" || atomic.LoadInt32(&embedCalls) != 1 {
		t.Fatalf("Embed() = %q calls=%d", marked, embedCalls)
	}
	extracted, err := client.ExtractWithReference(context.Background(), []byte("marked"), []byte("reference"), len(payload), options)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if string(extracted) != string(payload) {
		t.Fatalf("Extract() = %q, want %q", extracted, payload)
	}
}

func TestHiddenWatermarkRemoteClientSkipTLSVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		_, _ = io.WriteString(w, `{"payload_b64":"AA==","payload_bytes":1}`)
	}))
	defer server.Close()

	strict := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{
		URL:     server.URL,
		Timeout: time.Second,
	})
	_, strictErr := strict.Extract(context.Background(), []byte("image"), 1, HiddenWatermarkRemoteOptions{})
	if strictErr == nil {
		t.Fatal("strict TLS client succeeded against self-signed server")
	}

	insecure := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{
		URL:           server.URL,
		Timeout:       time.Second,
		SkipTLSVerify: true,
	})
	got, err := insecure.Extract(context.Background(), []byte("image"), 1, HiddenWatermarkRemoteOptions{})
	if err != nil {
		t.Fatalf("insecure Extract() error = %v", err)
	}
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("insecure Extract() = %x, want 00", got)
	}
}

func TestHiddenWatermarkRemoteClientHTTPErrorIncludesBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "remote failed", http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{URL: server.URL})
	_, err := client.Embed(context.Background(), []byte("image"), []byte("payload"), HiddenWatermarkRemoteOptions{})
	if err == nil || !strings.Contains(err.Error(), "remote failed") {
		t.Fatalf("Embed() error = %v, want body text", err)
	}
}

func TestHiddenWatermarkRemoteClientClassifiesGateway554AsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream timeout", 554)
	}))
	defer server.Close()

	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{URL: server.URL})
	_, err := client.Extract(context.Background(), []byte("image"), 8, HiddenWatermarkRemoteOptions{})
	if !errors.Is(err, ErrHiddenWatermarkRemoteTimeout) {
		t.Fatalf("Extract() error = %v, want ErrHiddenWatermarkRemoteTimeout", err)
	}
}

func TestHiddenWatermarkRemoteClientClassifiesStreamUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/watermark/embed-stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"type":"error","error":"`+ErrHiddenWatermarkRemoteUnavailable.Error()+`"}`+"\n")
	}))
	defer server.Close()

	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{URL: server.URL})
	_, err := client.EmbedWithProgress(
		context.Background(),
		[]byte("image"),
		[]byte("payload"),
		HiddenWatermarkRemoteOptions{},
		func(HiddenWatermarkProgress) {},
	)
	if !errors.Is(err, ErrHiddenWatermarkRemoteUnavailable) {
		t.Fatalf("EmbedWithProgress() error = %v, want ErrHiddenWatermarkRemoteUnavailable", err)
	}
}

func TestHiddenWatermarkRemoteClientStreamsProgressAndResult(t *testing.T) {
	payload := []byte("trace-id")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/watermark/extract-stream" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = io.WriteString(w, `{"type":"progress","stage":"extracting","percent":50,"completed":1,"total":2,"elapsed_ms":125}`+"\n")
		_, _ = io.WriteString(w, `{"type":"heartbeat","stage":"extracting","percent":50,"completed":1,"total":2,"elapsed_ms":2125}`+"\n")
		_, _ = io.WriteString(w, `{"type":"result","payload_b64":"`+base64.StdEncoding.EncodeToString(payload)+`","payload_bytes":8}`+"\n")
	}))
	defer server.Close()

	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{
		URL:     server.URL,
		Timeout: time.Second,
	})
	var updates []HiddenWatermarkProgress
	groups, err := client.ExtractCandidateGroupsWithReferenceProgress(
		context.Background(),
		[]byte("image"),
		nil,
		[]int{8},
		HiddenWatermarkRemoteOptions{},
		func(progress HiddenWatermarkProgress) { updates = append(updates, progress) },
	)
	if err != nil {
		t.Fatalf("ExtractCandidateGroupsWithReferenceProgress() error = %v", err)
	}
	if got := string(groups[8][0]); got != string(payload) {
		t.Fatalf("stream payload = %q, want %q", got, payload)
	}
	if len(updates) != 4 ||
		updates[0].Stage != "connecting_watermark_server" ||
		updates[0].Source != "gateway" ||
		updates[1].Stage != "watermark_server_connected" ||
		updates[1].Source != "gateway" ||
		updates[2].Completed != 1 ||
		updates[2].Total != 2 ||
		updates[2].Source != "engine" ||
		!updates[3].Heartbeat {
		t.Fatalf("stream progress = %+v", updates)
	}
}

func TestHiddenWatermarkRemoteClientUsesIndependentTLSConfig(t *testing.T) {
	client := NewHiddenWatermarkRemoteClient(HiddenWatermarkRemoteClientConfig{
		URL:           "https://watermark.internal",
		SkipTLSVerify: true,
	})
	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T", client.client.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("remote watermark transport did not enable skip TLS verify")
	}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	if defaultTransport.TLSClientConfig != nil && defaultTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("default HTTP transport was modified")
	}
	transport.TLSClientConfig = &tls.Config{}
}

func assertRemoteWatermarkFormOptions(t *testing.T, r *http.Request, want HiddenWatermarkRemoteOptions) {
	t.Helper()
	assertField := func(name string, want string) {
		if got := r.FormValue(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	assertField("password_wm", strconv.Itoa(want.PasswordWM))
	assertField("password_img", strconv.Itoa(want.PasswordImg))
	if want.D1 > 0 {
		assertField("d1", strconv.Itoa(want.D1))
		assertField("d2", strconv.Itoa(want.D2))
	}
	if want.OperationTimeoutSeconds > 0 {
		assertField("operation_timeout_seconds", strconv.Itoa(want.OperationTimeoutSeconds))
	}
	if want.Engine != "" {
		assertField("engine", want.Engine)
	}
	if want.DWTDctSVDRepeat > 0 {
		assertField("dwt_dct_svd_repeat", strconv.Itoa(want.DWTDctSVDRepeat))
	}
	if want.DWTDctSVDScale > 0 {
		assertField("dwt_dct_svd_scale", strconv.Itoa(want.DWTDctSVDScale))
	}
	if want.WatermarkWidth > 0 {
		assertField("watermark_width", strconv.Itoa(want.WatermarkWidth))
	}
	if want.WatermarkHeight > 0 {
		assertField("watermark_height", strconv.Itoa(want.WatermarkHeight))
	}
	if len(want.RecoverDimensions) > 0 {
		values := make([]string, 0, len(want.RecoverDimensions))
		for _, dimension := range want.RecoverDimensions {
			values = append(values, strconv.Itoa(dimension[0])+"x"+strconv.Itoa(dimension[1]))
		}
		assertField("recover_dimensions", strings.Join(values, ","))
	}
}
