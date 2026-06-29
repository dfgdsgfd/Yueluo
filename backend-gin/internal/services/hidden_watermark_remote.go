package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HiddenWatermarkRemoteClientConfig struct {
	URL           string
	APIKey        string
	Timeout       time.Duration
	SkipTLSVerify bool
}

type HiddenWatermarkRemoteOptions struct {
	PasswordWM              int
	PasswordImg             int
	D1                      int
	D2                      int
	Profile                 string
	OperationTimeoutSeconds int
	Engine                  string
	DWTDctSVDRepeat         int
	DWTDctSVDScale          int
	WatermarkWidth          int
	WatermarkHeight         int
	RecoverDimensions       [][2]int
}

var (
	ErrHiddenWatermarkRemoteTimeout     = errors.New("error.hidden_watermark_remote_timeout")
	ErrHiddenWatermarkRemoteUnavailable = errors.New("error.hidden_watermark_remote_unavailable")
	ErrHiddenWatermarkRemoteRejected    = errors.New("error.hidden_watermark_remote_rejected")
)

type HiddenWatermarkProgress struct {
	Stage     string `json:"stage"`
	Percent   int    `json:"percent"`
	Completed int    `json:"completed,omitempty"`
	Total     int    `json:"total,omitempty"`
	ElapsedMS int64  `json:"elapsedMs,omitempty"`
	Heartbeat bool   `json:"heartbeat,omitempty"`
	Source    string `json:"source,omitempty"`
	Profile   string `json:"profile,omitempty"`
}

type HiddenWatermarkProgressFunc func(HiddenWatermarkProgress)

type HiddenWatermarkRemoteClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewHiddenWatermarkRemoteClient(cfg HiddenWatermarkRemoteClientConfig) *HiddenWatermarkRemoteClient {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if baseURL == "" {
		return nil
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	client := &http.Client{Transport: transport}
	if cfg.Timeout > 0 {
		client.Timeout = cfg.Timeout
	}
	return &HiddenWatermarkRemoteClient{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		client:  client,
	}
}

func (c *HiddenWatermarkRemoteClient) Configured() bool {
	return c != nil && strings.TrimSpace(c.baseURL) != ""
}

func (c *HiddenWatermarkRemoteClient) Embed(ctx context.Context, imageData []byte, payload []byte, options HiddenWatermarkRemoteOptions) ([]byte, error) {
	return c.EmbedWithProgress(ctx, imageData, payload, options, nil)
}

func (c *HiddenWatermarkRemoteClient) EmbedWithProgress(ctx context.Context, imageData []byte, payload []byte, options HiddenWatermarkRemoteOptions, progress HiddenWatermarkProgressFunc) ([]byte, error) {
	if !c.Configured() {
		return nil, errors.New("hidden watermark remote service is not configured")
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
		Stage: "connecting_watermark_server", Percent: 2, Source: "gateway",
	})
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writeMultipartFile(writer, "file", "image.png", "image/png", imageData); err != nil {
		return nil, err
	}
	if err := writer.WriteField("payload_b64", base64.StdEncoding.EncodeToString(payload)); err != nil {
		return nil, err
	}
	if err := writeRemoteWatermarkOptions(writer, options); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	path := "/v1/watermark/embed"
	if progress != nil {
		path = "/v1/watermark/embed-stream"
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, normalizeRemoteWatermarkRequestError(err)
	}
	defer resp.Body.Close()
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
		Stage: "watermark_server_connected", Percent: 8, Source: "gateway",
	})
	if progress != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed) {
		reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
			Stage: "watermark_stream_unavailable", Percent: 10, Source: "gateway",
		})
		return c.Embed(ctx, imageData, payload, options)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, remoteWatermarkHTTPError(resp)
	}
	if progress != nil {
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxDecodedImagePixels*2))
		scanner.Buffer(make([]byte, 64<<10), maxDecodedImagePixels*2)
		for scanner.Scan() {
			var event remoteWatermarkStreamEvent
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				return nil, fmt.Errorf("decode hidden watermark progress: %w", err)
			}
			switch event.Type {
			case "progress", "heartbeat":
				progress(HiddenWatermarkProgress{
					Stage: event.Stage, Percent: event.Percent, Completed: event.Completed,
					Total: event.Total, ElapsedMS: event.ElapsedMS, Heartbeat: event.Type == "heartbeat",
					Source: "engine",
				})
			case "error":
				return nil, remoteWatermarkStreamError(event.Error, "error.watermark_embed_failed")
			case "result":
				data, err := base64.StdEncoding.DecodeString(event.ImageB64)
				if err != nil {
					return nil, err
				}
				if len(data) == 0 {
					return nil, errors.New("hidden watermark remote service returned empty image")
				}
				return data, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, normalizeRemoteWatermarkRequestError(err)
		}
		return nil, errors.New("hidden watermark remote stream ended without a result")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDecodedImagePixels))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("hidden watermark remote service returned empty image")
	}
	return data, nil
}

func (c *HiddenWatermarkRemoteClient) Extract(ctx context.Context, imageData []byte, payloadBytes int, options HiddenWatermarkRemoteOptions) ([]byte, error) {
	return c.ExtractWithReference(ctx, imageData, nil, payloadBytes, options)
}

func (c *HiddenWatermarkRemoteClient) ExtractWithReference(ctx context.Context, imageData, referenceData []byte, payloadBytes int, options HiddenWatermarkRemoteOptions) ([]byte, error) {
	payloads, err := c.ExtractCandidatesWithReference(ctx, imageData, referenceData, payloadBytes, options)
	if err != nil {
		return nil, err
	}
	return payloads[0], nil
}

func (c *HiddenWatermarkRemoteClient) ExtractCandidatesWithReference(ctx context.Context, imageData, referenceData []byte, payloadBytes int, options HiddenWatermarkRemoteOptions) ([][]byte, error) {
	groups, err := c.ExtractCandidateGroupsWithReference(ctx, imageData, referenceData, []int{payloadBytes}, options)
	if err != nil {
		return nil, err
	}
	payloads := groups[payloadBytes]
	if len(payloads) == 0 {
		return nil, errors.New("hidden watermark remote service returned no payload candidates")
	}
	return payloads, nil
}

func (c *HiddenWatermarkRemoteClient) ExtractCandidateGroupsWithReference(ctx context.Context, imageData, referenceData []byte, payloadBytesCandidates []int, options HiddenWatermarkRemoteOptions) (map[int][][]byte, error) {
	body, contentType, err := buildRemoteWatermarkExtractBody(imageData, referenceData, payloadBytesCandidates, options)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/watermark/extract", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, normalizeRemoteWatermarkRequestError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, remoteWatermarkHTTPError(resp)
	}
	var parsed remoteWatermarkExtractResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&parsed); err != nil {
		return nil, err
	}
	return decodeRemoteWatermarkGroups(parsed)
}

func (c *HiddenWatermarkRemoteClient) ExtractCandidateGroupsWithReferenceProgress(
	ctx context.Context,
	imageData, referenceData []byte,
	payloadBytesCandidates []int,
	options HiddenWatermarkRemoteOptions,
	progress HiddenWatermarkProgressFunc,
) (map[int][][]byte, error) {
	if !c.Configured() {
		return nil, errors.New("hidden watermark remote service is not configured")
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
		Stage: "connecting_watermark_server", Percent: 2, Source: "gateway",
	})
	body, contentType, err := buildRemoteWatermarkExtractBody(imageData, referenceData, payloadBytesCandidates, options)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/watermark/extract-stream", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/x-ndjson")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, normalizeRemoteWatermarkRequestError(err)
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
		Stage: "watermark_server_connected", Percent: 8, Source: "gateway",
	})
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		_ = resp.Body.Close()
		if progress != nil {
			progress(HiddenWatermarkProgress{Stage: "extracting", Percent: 20})
		}
		return c.ExtractCandidateGroupsWithReference(ctx, imageData, referenceData, payloadBytesCandidates, options)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, remoteWatermarkHTTPError(resp)
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 4<<20))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var event remoteWatermarkStreamEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode hidden watermark progress: %w", err)
		}
		switch event.Type {
		case "progress", "heartbeat":
			if progress != nil {
				progress(HiddenWatermarkProgress{
					Stage:     event.Stage,
					Percent:   event.Percent,
					Completed: event.Completed,
					Total:     event.Total,
					ElapsedMS: event.ElapsedMS,
					Heartbeat: event.Type == "heartbeat",
					Source:    "engine",
				})
			}
		case "error":
			return nil, remoteWatermarkStreamError(event.Error, "error.watermark_extract_failed")
		case "result":
			return decodeRemoteWatermarkGroups(event.response())
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, normalizeRemoteWatermarkRequestError(err)
	}
	return nil, errors.New("hidden watermark remote stream ended without a result")
}

func buildRemoteWatermarkExtractBody(
	imageData, referenceData []byte,
	payloadBytesCandidates []int,
	options HiddenWatermarkRemoteOptions,
) (*bytes.Buffer, string, error) {
	if len(payloadBytesCandidates) == 0 {
		return nil, "", errors.New("hidden watermark payload size is invalid")
	}
	sizeValues := make([]string, 0, len(payloadBytesCandidates))
	for _, payloadBytes := range payloadBytesCandidates {
		if payloadBytes <= 0 {
			return nil, "", errors.New("hidden watermark payload size is invalid")
		}
		sizeValues = append(sizeValues, strconv.Itoa(payloadBytes))
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writeMultipartFile(writer, "file", "image", "application/octet-stream", imageData); err != nil {
		return nil, "", err
	}
	if len(referenceData) > 0 {
		if err := writeMultipartFile(writer, "reference_file", "reference", "application/octet-stream", referenceData); err != nil {
			return nil, "", err
		}
	}
	if err := writer.WriteField("payload_bytes_candidates", strings.Join(sizeValues, ",")); err != nil {
		return nil, "", err
	}
	if err := writeRemoteWatermarkOptions(writer, options); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}

type remoteWatermarkExtractResponse struct {
	PayloadB64           string   `json:"payload_b64"`
	PayloadCandidatesB64 []string `json:"payload_candidates_b64"`
	PayloadBytes         int      `json:"payload_bytes"`
	PayloadGroups        []struct {
		PayloadBytes         int      `json:"payload_bytes"`
		PayloadCandidatesB64 []string `json:"payload_candidates_b64"`
	} `json:"payload_groups"`
}

type remoteWatermarkStreamEvent struct {
	Type                 string   `json:"type"`
	Stage                string   `json:"stage"`
	Percent              int      `json:"percent"`
	Completed            int      `json:"completed"`
	Total                int      `json:"total"`
	ElapsedMS            int64    `json:"elapsed_ms"`
	Error                string   `json:"error"`
	ImageB64             string   `json:"image_b64"`
	PayloadB64           string   `json:"payload_b64"`
	PayloadCandidatesB64 []string `json:"payload_candidates_b64"`
	PayloadBytes         int      `json:"payload_bytes"`
	PayloadGroups        []struct {
		PayloadBytes         int      `json:"payload_bytes"`
		PayloadCandidatesB64 []string `json:"payload_candidates_b64"`
	} `json:"payload_groups"`
}

func (event remoteWatermarkStreamEvent) response() remoteWatermarkExtractResponse {
	return remoteWatermarkExtractResponse{
		PayloadB64:           event.PayloadB64,
		PayloadCandidatesB64: event.PayloadCandidatesB64,
		PayloadBytes:         event.PayloadBytes,
		PayloadGroups:        event.PayloadGroups,
	}
}

func decodeRemoteWatermarkGroups(parsed remoteWatermarkExtractResponse) (map[int][][]byte, error) {
	encodedGroups := map[int][]string{}
	for _, group := range parsed.PayloadGroups {
		encodedGroups[group.PayloadBytes] = group.PayloadCandidatesB64
	}
	if len(encodedGroups) == 0 {
		encodedCandidates := parsed.PayloadCandidatesB64
		if len(encodedCandidates) == 0 && parsed.PayloadB64 != "" {
			encodedCandidates = []string{parsed.PayloadB64}
		}
		if parsed.PayloadBytes > 0 {
			encodedGroups[parsed.PayloadBytes] = encodedCandidates
		}
	}
	groups := map[int][][]byte{}
	for payloadBytes, encodedCandidates := range encodedGroups {
		payloads := make([][]byte, 0, len(encodedCandidates))
		for _, encoded := range encodedCandidates {
			payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
			if err != nil {
				return nil, err
			}
			if len(payload) != payloadBytes {
				return nil, fmt.Errorf("hidden watermark remote payload size mismatch: got %d want %d", len(payload), payloadBytes)
			}
			payloads = append(payloads, payload)
		}
		if len(payloads) > 0 {
			groups[payloadBytes] = payloads
		}
	}
	if len(groups) == 0 {
		return nil, errors.New("hidden watermark remote service returned no payload candidates")
	}
	return groups, nil
}

func writeRemoteWatermarkOptions(writer *multipart.Writer, options HiddenWatermarkRemoteOptions) error {
	fields := map[string]int{
		"password_wm":  options.PasswordWM,
		"password_img": options.PasswordImg,
	}
	if options.D1 > 0 {
		fields["d1"] = options.D1
		fields["d2"] = max(0, options.D2)
	}
	if options.OperationTimeoutSeconds > 0 {
		fields["operation_timeout_seconds"] = options.OperationTimeoutSeconds
	}
	if options.DWTDctSVDRepeat > 0 {
		fields["dwt_dct_svd_repeat"] = options.DWTDctSVDRepeat
	}
	if options.DWTDctSVDScale > 0 {
		fields["dwt_dct_svd_scale"] = options.DWTDctSVDScale
	}
	if options.WatermarkWidth > 0 {
		fields["watermark_width"] = options.WatermarkWidth
	}
	if options.WatermarkHeight > 0 {
		fields["watermark_height"] = options.WatermarkHeight
	}
	for key, value := range fields {
		if err := writer.WriteField(key, strconv.Itoa(value)); err != nil {
			return err
		}
	}
	if engine := strings.TrimSpace(options.Engine); engine != "" {
		if err := writer.WriteField("engine", engine); err != nil {
			return err
		}
	}
	if len(options.RecoverDimensions) > 0 {
		values := make([]string, 0, len(options.RecoverDimensions))
		for _, dimension := range options.RecoverDimensions {
			if dimension[0] > 0 && dimension[1] > 0 {
				values = append(values, fmt.Sprintf("%dx%d", dimension[0], dimension[1]))
			}
		}
		if len(values) > 0 {
			if err := writer.WriteField("recover_dimensions", strings.Join(values, ",")); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *HiddenWatermarkRemoteClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	endpoint, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-Internal-API-Key", c.apiKey)
	}
	return req, nil
}

func writeMultipartFile(writer *multipart.Writer, fieldName, filename, contentType string, data []byte) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

func remoteWatermarkHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	switch resp.StatusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusBadGateway,
		http.StatusGatewayTimeout, 524, 554:
		return fmt.Errorf("%w: upstream status %d: %s", ErrHiddenWatermarkRemoteTimeout, resp.StatusCode, message)
	case http.StatusServiceUnavailable:
		return fmt.Errorf("%w: upstream status %d: %s", ErrHiddenWatermarkRemoteUnavailable, resp.StatusCode, message)
	}
	return fmt.Errorf("%w: upstream status %d: %s", ErrHiddenWatermarkRemoteRejected, resp.StatusCode, message)
}

func remoteWatermarkStreamError(raw, fallback string) error {
	message := firstNonEmptyImageString(strings.TrimSpace(raw), fallback)
	switch message {
	case ErrHiddenWatermarkRemoteUnavailable.Error():
		return ErrHiddenWatermarkRemoteUnavailable
	case ErrHiddenWatermarkRemoteTimeout.Error():
		return ErrHiddenWatermarkRemoteTimeout
	case ErrHiddenWatermarkRemoteRejected.Error():
		return ErrHiddenWatermarkRemoteRejected
	}
	return errors.New(message)
}

func normalizeRemoteWatermarkRequestError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrHiddenWatermarkRemoteTimeout, err)
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return fmt.Errorf("%w: %v", ErrHiddenWatermarkRemoteTimeout, err)
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("%w: %v", ErrHiddenWatermarkRemoteUnavailable, err)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Errorf("%w: %v", ErrHiddenWatermarkRemoteUnavailable, err)
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "connection refused") ||
		strings.Contains(message, "actively refused") ||
		strings.Contains(message, "no such host") ||
		strings.Contains(message, "connection reset") {
		return fmt.Errorf("%w: %v", ErrHiddenWatermarkRemoteUnavailable, err)
	}
	return err
}
