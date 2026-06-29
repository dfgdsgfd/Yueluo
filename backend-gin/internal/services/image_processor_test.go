package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/disintegration/imaging"
	avif "github.com/gen2brain/avif"
	webp "github.com/gen2brain/webp"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
	xwebp "golang.org/x/image/webp"

	"yuem-go/backend-gin/internal/domain"
)

const (
	testTraceToken = "0102030405060708"
	testShortCode  = "a1b2c3d4"
)

func configureTestShortCodeResolver(processor *ImageProcessor, traceToken, shortCode string) {
	processor.SetPayloadResolver(func(_ context.Context, payload []byte) HiddenWatermarkData {
		if hex.EncodeToString(payload) != shortCode {
			return HiddenWatermarkData{Found: false}
		}
		return HiddenWatermarkData{
			Found:         true,
			Valid:         true,
			Version:       3,
			TraceToken:    traceToken,
			TraceResolved: true,
			PayloadBytes:  len(payload),
			PayloadBits:   len(payload) * 8,
			PayloadFormat: "short_code_v1",
		}
	})
}

func TestImageProcessorConvertsAndExtractsHiddenWatermark(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "image_webp_quality", 85)
	_ = settings.Set(context.Background(), "hidden_watermark_custom_text", "yuem")
	_ = settings.Set(context.Background(), "hidden_watermark_include_custom", true)
	_ = settings.Set(context.Background(), "hidden_watermark_include_username", true)

	source := gradientImage(512, 512)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}

	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "photo.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
		User: &RequestUser{
			ID:       42,
			UserID:   "public-user",
			Nickname: "测试用户",
		},
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Format != "webp" || result.ContentType != "image/webp" || result.Filename != "photo.webp" {
		t.Fatalf("unexpected processed image metadata: format=%q contentType=%q filename=%q", result.Format, result.ContentType, result.Filename)
	}
	if !result.WatermarkApplied {
		decoded, decodeErr := xwebp.Decode(bytes.NewReader(result.Data))
		if decodeErr != nil {
			t.Fatalf("watermark was not applied and output decode failed: warning=%q err=%v", result.WatermarkWarning, decodeErr)
		}
		raw, extractErr := watermark.Extract(
			context.Background(),
			decoded,
			mark.NewExtract(domain.ImageWatermarkShortCodeBytes*8),
			watermark.WithBlockShape(hiddenWatermarkBlockWidth, hiddenWatermarkBlockHeight),
			watermark.WithD1D2(hiddenWatermarkD1, hiddenWatermarkD2),
		)
		if extractErr != nil {
			t.Fatalf("watermark was not applied: warning=%q extract=%v", result.WatermarkWarning, extractErr)
		}
		t.Fatalf("watermark was not applied: warning=%q raw=%x", result.WatermarkWarning, raw.DecodeToBytes())
	}
	if len(result.Data) == 0 || result.ProcessedSize != int64(len(result.Data)) {
		t.Fatalf("invalid processed image size: processed=%d actual=%d", result.ProcessedSize, len(result.Data))
	}

	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid {
		t.Fatalf("watermark not found or invalid: %+v", extracted)
	}
	if extracted.TraceToken == "" || extracted.TraceToken != result.WatermarkTraceToken ||
		extracted.PayloadBytes != domain.ImageWatermarkShortCodeBytes || extracted.PayloadBits != 32 || extracted.PayloadFormat != "short_code_v1" {
		t.Fatalf("unexpected extracted watermark: %+v", extracted)
	}
}

func TestImageProcessorLocalEngineRecoversQQStyleScreenshotWithReference(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "local")
	_ = settings.Set(context.Background(), "hidden_watermark_profile", "robust")
	processor := NewImageProcessor(settings, "screenshot-watermark-secret", 10<<20, nil)
	const traceToken = "0102030405060708"
	configureTestShortCodeResolver(processor, traceToken, testShortCode)

	var sourceData bytes.Buffer
	if err := jpeg.Encode(&sourceData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	marked, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  sourceData.Bytes(),
		Filename:              "source.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		ForceWatermark:        true,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !marked.WatermarkApplied {
		t.Fatalf("watermark was not applied: %+v", marked)
	}
	markedImage, err := xwebp.Decode(bytes.NewReader(marked.Data))
	if err != nil {
		t.Fatalf("decode marked image: %v", err)
	}
	qqPreview := imaging.Resize(markedImage, 448, 448, imaging.Lanczos)
	canvas := imaging.New(576, 608, color.NRGBA{R: 245, G: 245, B: 245, A: 255})
	canvas = imaging.Paste(canvas, qqPreview, image.Pt(64, 80))
	var screenshot bytes.Buffer
	if err := jpeg.Encode(&screenshot, canvas, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatalf("encode screenshot: %v", err)
	}
	extracted, err := processor.ExtractWithReference(context.Background(), screenshot.Bytes(), marked.Data)
	if err != nil {
		t.Fatalf("ExtractWithReference() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid || extracted.TraceToken != traceToken || extracted.WatermarkEngine != "local" {
		t.Fatalf("local screenshot extraction mismatch: %+v", extracted)
	}
}

func TestShortCodeWatermarkRequiresDatabaseResolution(t *testing.T) {
	processor := NewImageProcessor(NewSettingsService(nil, nil), "token-secret", 10<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	valid := processor.decodeExtractedWatermarkPayload(context.Background(), []byte{0xa1, 0xb2, 0xc3, 0xd4}, domain.ImageWatermarkShortCodeBytes)
	if !valid.Found || !valid.Valid || !valid.TraceResolved || valid.TraceToken != "0102030405060708" {
		t.Fatalf("valid token result = %+v", valid)
	}
	invalid := processor.decodeExtractedWatermarkPayload(context.Background(), []byte{0xa1, 0xb2, 0xc3, 0xd5}, domain.ImageWatermarkShortCodeBytes)
	if invalid.Found || invalid.Valid || invalid.TraceResolved {
		t.Fatalf("bit-flipped token must be unresolved: %+v", invalid)
	}
}

func TestWatermarkLibraryDirectRoundTrip(t *testing.T) {
	source := gradientImage(512, 512)
	payload := make([]byte, domain.ImageWatermarkShortCodeBytes)
	for index := range payload {
		payload[index] = byte(index*37 + 11)
	}
	marked, err := watermark.Embed(
		context.Background(),
		source,
		mark.NewBytes(payload),
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	)
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	decoder, err := watermark.Extract(
		context.Background(),
		imaging.Clone(marked),
		mark.NewExtract(domain.ImageWatermarkShortCodeBytes*8),
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if got := decoder.DecodeToBytes(); !bytes.Equal(got, payload) {
		t.Fatalf("direct watermark round trip mismatch: got %x want %x", got, payload)
	}

	var encoded bytes.Buffer
	if err := webp.Encode(&encoded, marked, webp.Options{Lossless: true, Quality: 85, Method: 4, Exact: true}); err != nil {
		t.Fatalf("webp encode: %v", err)
	}
	decoded, err := xwebp.Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("webp decode: %v", err)
	}
	decoder, err = watermark.Extract(
		context.Background(),
		decoded,
		mark.NewExtract(domain.ImageWatermarkShortCodeBytes*8),
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	)
	if err != nil {
		t.Fatalf("Extract() after webp error = %v", err)
	}
	if got := decoder.DecodeToBytes(); !bytes.Equal(got, payload) {
		t.Fatalf("webp watermark round trip mismatch: got %x want %x", got, payload)
	}
}

func TestImageProcessorProtectionWatermarkUsesToken(t *testing.T) {
	source := gradientImage(512, 512)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessor(NewSettingsService(nil, nil), "protection-secret", 10<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "protected.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "viewer-account", Nickname: "观看用户"},
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !result.WatermarkApplied {
		t.Fatalf("structured protection watermark was not applied: %s", result.WatermarkWarning)
	}
	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid {
		t.Fatalf("structured protection watermark invalid: %+v", extracted)
	}
	if extracted.TraceToken == "" || extracted.TraceToken != result.WatermarkTraceToken ||
		extracted.PayloadBytes != domain.ImageWatermarkShortCodeBytes {
		t.Fatalf("unexpected structured protection watermark: %+v", extracted)
	}
}

func TestImageProcessorProtectionWatermarkPreservesLargeDimensions(t *testing.T) {
	const width, height = 2048, 2048
	source := gradientImage(width, height)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessor(NewSettingsService(nil, nil), "protection-secret", 20<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "large-protected.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "viewer-account"},
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Width != width || result.Height != height {
		t.Fatalf("processed dimensions = %dx%d, want %dx%d", result.Width, result.Height, width, height)
	}
	if !result.WatermarkApplied {
		t.Fatalf("large protection watermark was not applied: %s", result.WatermarkWarning)
	}
	decodedSource, err := jpeg.Decode(bytes.NewReader(jpegData.Bytes()))
	if err != nil {
		t.Fatalf("decode source jpeg: %v", err)
	}
	decodedResult, err := xwebp.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("decode protected webp: %v", err)
	}
	if difference := meanAbsoluteRGBDifference(decodedSource, decodedResult); difference > 1.5 {
		t.Fatalf("mean absolute RGB difference = %.3f, want <= 1.5", difference)
	}
	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid || extracted.TraceToken != result.WatermarkTraceToken {
		t.Fatalf("large protection watermark invalid: %+v", extracted)
	}
}

func meanAbsoluteRGBDifference(left, right image.Image) float64 {
	bounds := left.Bounds().Intersect(right.Bounds())
	if bounds.Empty() {
		return 255
	}
	var total uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			leftR, leftG, leftB, _ := left.At(x, y).RGBA()
			rightR, rightG, rightB, _ := right.At(x, y).RGBA()
			total += uint64(absInt(int(leftR>>8) - int(rightR>>8)))
			total += uint64(absInt(int(leftG>>8) - int(rightG>>8)))
			total += uint64(absInt(int(leftB>>8) - int(rightB>>8)))
		}
	}
	return float64(total) / float64(bounds.Dx()*bounds.Dy()*3)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func TestImageProcessorNeverWatermarksProfileImages(t *testing.T) {
	source := gradientImage(512, 512)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessor(NewSettingsService(nil, nil), "profile-watermark-secret", 10<<20, nil)
	for _, purpose := range []ImagePurpose{ImagePurposeAvatar, ImagePurposeBackground} {
		t.Run(string(purpose), func(t *testing.T) {
			result, err := processor.Process(context.Background(), ProcessImageInput{
				Data:           jpegData.Bytes(),
				Filename:       string(purpose) + ".jpg",
				ContentType:    "image/jpeg",
				Purpose:        purpose,
				User:           &RequestUser{ID: 42, UserID: "profile-owner"},
				ForceWatermark: true,
			})
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if result.WatermarkApplied || result.WatermarkWarning != "" {
				t.Fatalf("profile image must not use hidden watermark: %+v", result)
			}
		})
	}
}

func TestImageProcessorUsesConfiguredHiddenWatermarkProfile(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_profile", "custom")
	_ = settings.Set(context.Background(), "hidden_watermark_block_width", 6)
	_ = settings.Set(context.Background(), "hidden_watermark_block_height", 4)
	_ = settings.Set(context.Background(), "hidden_watermark_coefficient_mode", "d1d2")
	_ = settings.Set(context.Background(), "hidden_watermark_d1", 21)
	_ = settings.Set(context.Background(), "hidden_watermark_d2", 9)
	_ = settings.Set(context.Background(), "hidden_watermark_ecc_mode", "golay")
	_ = settings.Set(context.Background(), "hidden_watermark_golay_seed", int(mark.DefaultShuffleSeed))
	processor := NewImageProcessor(settings, "custom-profile-secret", 10<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	profiles := processor.hiddenWatermarkProfiles()
	if len(profiles) == 0 || profiles[0] != (hiddenWatermarkProfile{blockWidth: 6, blockHeight: 4, coefficientMode: "d1d2", d1: 21, d2: 9, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed}) {
		t.Fatalf("configured profile was not first: %#v", profiles)
	}

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "custom-profile.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "profile-owner"},
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !result.WatermarkApplied || !extracted.Found || !extracted.Valid || extracted.TraceToken != result.WatermarkTraceToken {
		t.Fatalf("configured profile watermark mismatch: result=%+v extracted=%+v", result, extracted)
	}
}

func TestImageProcessorRemoteEngineUsesRemoteService(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "remote")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_password_wm", 77)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_password_img", 88)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_operation_timeout_seconds", 66)
	var embeddedPayload []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatal(err)
		}
		assertRemoteWatermarkFormOptions(t, r, HiddenWatermarkRemoteOptions{
			PasswordWM:              77,
			PasswordImg:             88,
			D1:                      18,
			D2:                      8,
			OperationTimeoutSeconds: 66,
		})
		switch r.URL.Path {
		case "/v1/watermark/embed":
			raw, err := base64.StdEncoding.DecodeString(r.FormValue("payload_b64"))
			if err != nil {
				t.Fatal(err)
			}
			embeddedPayload = append([]byte(nil), raw...)
			var out bytes.Buffer
			if err := png.Encode(&out, gradientImage(512, 512)); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(out.Bytes())
		case "/v1/watermark/extract":
			invalid := make([]byte, len(embeddedPayload))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload_b64": base64.StdEncoding.EncodeToString(invalid),
				"payload_candidates_b64": []string{
					base64.StdEncoding.EncodeToString(invalid),
					base64.StdEncoding.EncodeToString(embeddedPayload),
				},
				"payload_bytes": len(embeddedPayload),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessorWithRemote(
		settings,
		"remote-secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: server.URL},
	)
	const traceToken = "0102030405060708"
	configureTestShortCodeResolver(processor, traceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "remote.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "remote-user"},
		ForceWatermark:        true,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !result.WatermarkApplied || len(embeddedPayload) != domain.ImageWatermarkShortCodeBytes {
		t.Fatalf("remote watermark not applied: result=%+v payload=%d", result, len(embeddedPayload))
	}
	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid || extracted.TraceToken != result.WatermarkTraceToken {
		t.Fatalf("remote extraction mismatch: %+v", extracted)
	}
}

func TestImageProcessorRemoteSelfVerificationSendsReference(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "remote")
	var embeddedPayload []byte
	extractSawReference := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/v1/watermark/embed":
			raw, err := base64.StdEncoding.DecodeString(r.FormValue("payload_b64"))
			if err != nil {
				t.Fatal(err)
			}
			embeddedPayload = append([]byte(nil), raw...)
			var out bytes.Buffer
			if err := png.Encode(&out, gradientImage(512, 512)); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(out.Bytes())
		case "/v1/watermark/extract":
			if file, _, err := r.FormFile("reference_file"); err == nil {
				extractSawReference = true
				_ = file.Close()
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload_b64":            base64.StdEncoding.EncodeToString(embeddedPayload),
				"payload_candidates_b64": []string{base64.StdEncoding.EncodeToString(embeddedPayload)},
				"payload_bytes":          len(embeddedPayload),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessorWithRemote(
		settings,
		"remote-secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: server.URL},
	)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	_, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "remote-reference.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
		VerifyWithReference:   true,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !extractSawReference {
		t.Fatal("remote self-verification did not send reference_file")
	}
}

func TestImageProcessorRemoteEngineDoesNotFallbackToLocal(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "remote")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_engine", "blind_watermark")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "remote unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessorWithRemote(
		settings,
		"remote-secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: server.URL},
	)
	_, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "remote-failed.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "remote-user"},
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if !errors.Is(err, ErrHiddenWatermarkRemoteUnavailable) {
		t.Fatalf("Process() error = %v, want ErrHiddenWatermarkRemoteUnavailable", err)
	}
	if processor.hiddenWatermarkAttemptCount() != len(hiddenWatermarkRemoteStrengthProfiles) {
		t.Fatalf("remote attempt count = %d, want %d", processor.hiddenWatermarkAttemptCount(), len(hiddenWatermarkRemoteStrengthProfiles))
	}
}

func TestRemoteWatermarkAdaptiveProfilesEscalateFromFidelityToOfficial(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_profile", "adaptive")
	processor := NewImageProcessor(settings, "secret", 10<<20, nil)
	profiles := processor.remoteWatermarkEmbedProfiles()
	if len(profiles) != 4 {
		t.Fatalf("profiles = %#v, want four adaptive levels", profiles)
	}
	want := []struct {
		name   string
		d1, d2 int
	}{
		{name: "fidelity", d1: 18, d2: 8},
		{name: "balanced", d1: 24, d2: 12},
		{name: "strong", d1: 30, d2: 15},
		{name: "official", d1: 36, d2: 20},
	}
	for index, expected := range want {
		if profiles[index].name != expected.name || profiles[index].d1 != expected.d1 || profiles[index].d2 != expected.d2 {
			t.Fatalf("profile[%d] = %#v, want %#v", index, profiles[index], expected)
		}
	}
}

func TestRemoteWatermarkPreferRobustSkipsFidelity(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_engine", "blind_watermark")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_profile", "adaptive")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_password_wm", 77)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_password_img", 88)
	processor := NewImageProcessorWithRemote(
		settings,
		"secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: "http://127.0.0.1:8090"},
	)
	input := ProcessImageInput{PreferRobust: true}
	profiles := processor.remoteWatermarkEmbedProfilesForInput(input)
	if len(profiles) != 3 {
		t.Fatalf("profiles = %#v, want three robust levels", profiles)
	}
	want := []struct {
		name   string
		d1, d2 int
	}{
		{name: "balanced", d1: 24, d2: 12},
		{name: "strong", d1: 30, d2: 15},
		{name: "official", d1: 36, d2: 20},
	}
	for index, expected := range want {
		if profiles[index].name != expected.name || profiles[index].d1 != expected.d1 || profiles[index].d2 != expected.d2 ||
			profiles[index].passwordWM != 77 || profiles[index].passwordImg != 88 {
			t.Fatalf("profile[%d] = %#v, want %#v with configured passwords", index, profiles[index], expected)
		}
	}
	if got := processor.hiddenWatermarkAttemptCountForInput(input); got != len(want) {
		t.Fatalf("hiddenWatermarkAttemptCountForInput() = %d, want %d", got, len(want))
	}
	options := processor.remoteWatermarkOptions(ProcessImageInput{WatermarkAttempt: 0, PreferRobust: true})
	if options.Profile != "balanced" || options.D1 != 24 || options.D2 != 12 || options.PasswordWM != 77 || options.PasswordImg != 88 {
		t.Fatalf("remoteWatermarkOptions(0, true) = %#v, want balanced with configured passwords", options)
	}
}

func TestRemoteWatermarkAutoEngineAppendsDWTAttempt(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_engine", "auto")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_profile", "adaptive")
	processor := NewImageProcessorWithRemote(
		settings,
		"secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: "http://127.0.0.1:8090"},
	)

	input := ProcessImageInput{PreferRobust: true}
	attempts := processor.remoteWatermarkEmbedAttemptsForInput(input)
	if len(attempts) != 4 {
		t.Fatalf("auto attempts = %#v, want 4 attempts", attempts)
	}
	for index, wantProfile := range []string{"balanced", "strong", "official"} {
		if attempts[index].engine != "blind_watermark" || attempts[index].profile.name != wantProfile {
			t.Fatalf("attempt[%d] = %#v, want blind_watermark/%s", index, attempts[index], wantProfile)
		}
	}
	if attempts[3].engine != "dwt_dct_svd" || attempts[3].profile.name != "official" {
		t.Fatalf("attempt[3] = %#v, want dwt_dct_svd/official", attempts[3])
	}
	if got := processor.hiddenWatermarkAttemptCountForInput(input); got != len(attempts) {
		t.Fatalf("hiddenWatermarkAttemptCountForInput() = %d, want %d", got, len(attempts))
	}
	if options := processor.remoteWatermarkOptions(ProcessImageInput{WatermarkAttempt: 3, PreferRobust: true}); options.Engine != "dwt_dct_svd" || options.Profile != "official" {
		t.Fatalf("remoteWatermarkOptions(attempt 3) = %#v, want dwt_dct_svd/official", options)
	}
}

func TestRemoteRequiredAttemptCountIgnoresGlobalLocalEngine(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "local")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_engine", "blind_watermark")
	_ = settings.Set(context.Background(), "hidden_watermark_remote_profile", "official")
	processor := NewImageProcessorWithRemote(
		settings,
		"secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: "http://127.0.0.1:8090"},
	)
	processor.RequireRemoteHiddenWatermark()
	got := processor.hiddenWatermarkAttemptCountForInput(ProcessImageInput{PreferRobust: true, ForceRemoteAdaptive: true})
	if got != 3 {
		t.Fatalf("hiddenWatermarkAttemptCountForInput() = %d, want 3 robust remote attempts", got)
	}
	profiles := processor.remoteWatermarkEmbedProfilesForInput(ProcessImageInput{PreferRobust: true, ForceRemoteAdaptive: true})
	if len(profiles) != 3 || profiles[0].name != "balanced" || profiles[1].name != "strong" || profiles[2].name != "official" {
		t.Fatalf("forced remote adaptive profiles = %#v, want balanced/strong/official", profiles)
	}
	for attempt, want := range []string{"balanced", "strong", "official"} {
		options := processor.remoteWatermarkOptions(ProcessImageInput{
			WatermarkAttempt:    attempt,
			PreferRobust:        true,
			ForceRemoteAdaptive: true,
		})
		if options.Profile != want {
			t.Fatalf("forced adaptive attempt %d = %q, want %q", attempt, options.Profile, want)
		}
	}
}

func TestRemoteWatermarkPreferRobustLiftsExplicitFidelity(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_profile", "fidelity")
	processor := NewImageProcessor(settings, "secret", 10<<20, nil)
	profiles := processor.remoteWatermarkEmbedProfilesForInput(ProcessImageInput{PreferRobust: true})
	if len(profiles) != 1 || profiles[0].name != "balanced" || profiles[0].d1 != 24 || profiles[0].d2 != 12 {
		t.Fatalf("profiles = %#v, want balanced", profiles)
	}
}

func TestImageProcessorAutoEngineFallsBackToLocal(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "auto")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "remote unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessorWithRemote(
		settings,
		"auto-secret",
		10<<20,
		nil,
		HiddenWatermarkRemoteClientConfig{URL: server.URL},
	)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "auto-fallback.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !result.WatermarkApplied || result.WatermarkEngine != "local" || result.WatermarkTraceToken != testTraceToken {
		t.Fatalf("auto fallback result = %+v", result)
	}
}

func TestImageProcessorUsesAuthorRecommendedPreset(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_profile", "author_recommended")
	processor := NewImageProcessor(settings, "author-profile-secret", 10<<20, nil)
	profiles := processor.hiddenWatermarkProfiles()
	want := hiddenWatermarkProfile{blockWidth: 8, blockHeight: 8, coefficientMode: "d1d2", d1: 21, d2: 9, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed}
	if len(profiles) == 0 || profiles[0] != want {
		t.Fatalf("author recommended profile = %#v, want first %#v", profiles, want)
	}
}

func TestProtectedWatermarkPrefersRobustProfile(t *testing.T) {
	processor := NewImageProcessor(NewSettingsService(nil, nil), "robust-profile-secret", 10<<20, nil)
	profiles := processor.hiddenWatermarkProfilesForInput(ProcessImageInput{PreferRobust: true})
	robust, _ := hiddenWatermarkPresetProfile("robust")
	if len(profiles) == 0 || profiles[0] != robust {
		t.Fatalf("protected profiles = %#v, want robust first", profiles)
	}
}

func TestImageProcessorExtractsWithConfiguredD1AndWithoutECC(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_profile", "custom")
	_ = settings.Set(context.Background(), "hidden_watermark_block_width", 8)
	_ = settings.Set(context.Background(), "hidden_watermark_block_height", 8)
	_ = settings.Set(context.Background(), "hidden_watermark_coefficient_mode", "d1")
	_ = settings.Set(context.Background(), "hidden_watermark_d1", 21)
	_ = settings.Set(context.Background(), "hidden_watermark_ecc_mode", "none")

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessor(settings, "without-ecc-secret", 10<<20, nil)
	configureTestShortCodeResolver(processor, testTraceToken, testShortCode)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  jpegData.Bytes(),
		Filename:              "without-ecc.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 24, UserID: "without-ecc-user"},
		ForceWatermark:        true,
		WatermarkTraceToken:   testTraceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	extracted, err := processor.Extract(context.Background(), result.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !result.WatermarkApplied || !extracted.Found || !extracted.Valid || extracted.TraceToken != result.WatermarkTraceToken {
		t.Fatalf("configured D1/WithoutECC extraction mismatch: result=%+v extracted=%+v", result, extracted)
	}
}

func TestImageProcessorUsesAvatarQualityAndCanDisableWatermark(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_enabled", false)
	_ = settings.Set(context.Background(), "image_avatar_webp_quality", 75)

	source := gradientImage(512, 512)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        jpegData.Bytes(),
		Filename:    "avatar.jpeg",
		ContentType: "image/jpeg",
		Purpose:     ImagePurposeAvatar,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.WatermarkApplied || result.Format != "webp" {
		t.Fatalf("unexpected avatar result: %+v", result)
	}
	contentResult, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        jpegData.Bytes(),
		Filename:    "content.jpeg",
		ContentType: "image/jpeg",
		Purpose:     ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("content Process() error = %v", err)
	}
	if result.ProcessedSize >= contentResult.ProcessedSize {
		t.Fatalf("avatar quality 75 size = %d, want less than content quality 85 size %d", result.ProcessedSize, contentResult.ProcessedSize)
	}
}

func TestImageProcessorRejectsMIMEAndInvalidData(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)

	source := gradientImage(32, 32)
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, source, nil); err != nil {
		t.Fatalf("encode jpeg fixture: %v", err)
	}
	if _, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        jpegData.Bytes(),
		Filename:    "fake.png",
		ContentType: "image/png",
	}); err != ErrImageMIME {
		t.Fatalf("MIME mismatch error = %v, want %v", err, ErrImageMIME)
	}
	if _, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        jpegData.Bytes(),
		Filename:    "legacy.jpg",
		ContentType: "image/jpg",
	}); err != nil {
		t.Fatalf("image/jpg alias should be accepted: %v", err)
	}
	if _, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        []byte("not an image"),
		Filename:    "bad.jpg",
		ContentType: "image/jpeg",
	}); err != ErrImageInvalid {
		t.Fatalf("invalid image error = %v, want %v", err, ErrImageInvalid)
	}
}

func TestImageProcessorAcceptsAVIFInputAndConvertsToWebP(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_enabled", false)
	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)

	source := gradientImage(40, 32)
	var avifData bytes.Buffer
	if err := avif.Encode(&avifData, source, avif.Options{Quality: 80, Speed: 10}); err != nil {
		t.Fatalf("encode avif fixture: %v", err)
	}

	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        avifData.Bytes(),
		Filename:    "photo.avif",
		ContentType: "image/avif",
		Purpose:     ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("Process() AVIF error = %v", err)
	}
	if result.Format != "webp" || result.ContentType != "image/webp" || result.Filename != "photo.webp" {
		t.Fatalf("unexpected AVIF output metadata: format=%q contentType=%q filename=%q", result.Format, result.ContentType, result.Filename)
	}
	decoded, err := xwebp.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("decode AVIF-derived webp: %v", err)
	}
	if got := decoded.Bounds(); got.Dx() != 40 || got.Dy() != 32 {
		t.Fatalf("decoded AVIF-derived bounds = %dx%d, want 40x32", got.Dx(), got.Dy())
	}
}

func TestImageProcessorRejectsAVIFMIMEAndAnimation(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)

	source := gradientImage(16, 16)
	var avifData bytes.Buffer
	if err := avif.Encode(&avifData, source, avif.Options{Quality: 80, Speed: 10}); err != nil {
		t.Fatalf("encode avif fixture: %v", err)
	}
	if _, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        avifData.Bytes(),
		Filename:    "fake.png",
		ContentType: "image/png",
	}); err != ErrImageMIME {
		t.Fatalf("AVIF MIME mismatch error = %v, want %v", err, ErrImageMIME)
	}

	animatedHeader := []byte{
		0, 0, 0, 24,
		'f', 't', 'y', 'p',
		'a', 'v', 'i', 's',
		0, 0, 0, 0,
		'a', 'v', 'i', 's',
		'a', 'v', 'i', 'f',
	}
	if _, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        animatedHeader,
		Filename:    "animated.avifs",
		ContentType: "image/avif",
	}); err != ErrImageAnimatedUnsupported {
		t.Fatalf("animated AVIF error = %v, want %v", err, ErrImageAnimatedUnsupported)
	}
}

func TestImageProcessorPreservesTransparencyAndResizes(t *testing.T) {
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_enabled", false)
	_ = settings.Set(context.Background(), "image_max_width", 64)
	_ = settings.Set(context.Background(), "image_max_height", 64)

	source := image.NewNRGBA(image.Rect(0, 0, 160, 80))
	for y := range 80 {
		for x := range 160 {
			source.SetNRGBA(x, y, color.NRGBA{R: 200, G: 80, B: 40, A: uint8((x * 255) / 159)})
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, source); err != nil {
		t.Fatalf("encode png fixture: %v", err)
	}

	processor := NewImageProcessor(settings, "test-watermark-secret", 10<<20, nil)
	result, err := processor.Process(context.Background(), ProcessImageInput{
		Data:        pngData.Bytes(),
		Filename:    "transparent.png",
		ContentType: "image/png",
		Purpose:     ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Width != 64 || result.Height != 32 {
		t.Fatalf("processed dimensions = %dx%d, want 64x32", result.Width, result.Height)
	}
	decoded, err := xwebp.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if _, _, _, alpha := decoded.At(0, 0).RGBA(); alpha >= 0xffff {
		t.Fatalf("expected transparent alpha at left edge, got %d", alpha)
	}
}

func gradientImage(width, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8(((x + y) * 255) / (width + height)),
				A: 255,
			})
		}
	}
	return img
}
