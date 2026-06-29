//go:build cgo

package services

import (
	"bytes"
	"context"
	"image/jpeg"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestLibvipsPipelineAutoRotatesAndKeepsSafeMetadata(t *testing.T) {
	if err := ensureLibvipsStarted(); err != nil {
		t.Fatalf("start libvips: %v", err)
	}
	sourceRef, err := vips.NewImageFromGoImage(gradientImage(48, 32))
	if err != nil {
		t.Fatalf("create source image: %v", err)
	}
	if err := sourceRef.TransformICCProfileWithFallback(vips.SRGBV2MicroICCProfilePath, vips.SRGBV2MicroICCProfilePath); err != nil {
		sourceRef.Close()
		t.Fatalf("attach sRGB profile: %v", err)
	}
	if err := sourceRef.SetOrientation(6); err != nil {
		sourceRef.Close()
		t.Fatalf("set orientation: %v", err)
	}
	sourceRef.SetString("exif-ifd0-Model", "TraceCam (TraceCam, ASCII, 9 components, 9 bytes)")
	sourceRef.SetString("exif-ifd0-Copyright", "Yuem (Yuem, ASCII, 5 components, 5 bytes)")
	sourceRef.SetString("exif-ifd2-BodySerialNumber", "camera-secret (camera-secret, ASCII, 14 components, 14 bytes)")
	sourceRef.SetString("exif-ifd3-GPSLatitude", "31.2304 (31.2304, ASCII, 8 components, 8 bytes)")
	input, _, err := sourceRef.ExportJpeg(vips.NewJpegExportParams())
	sourceRef.Close()
	if err != nil {
		t.Fatalf("export source jpeg: %v", err)
	}

	img, metadata, err := decodeWithColorManagement(input)
	if err != nil {
		t.Fatalf("decodeWithColorManagement() error = %v", err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 48 {
		t.Fatalf("autorotated dimensions = %dx%d, want 32x48", img.Bounds().Dx(), img.Bounds().Dy())
	}
	if len(metadata.ICCProfile) == 0 {
		t.Fatal("normalized sRGB ICC profile is missing")
	}
	if metadata.SafeEXIF["exif-ifd0-Model"] == "" || metadata.SafeEXIF["exif-ifd0-Copyright"] == "" {
		t.Fatalf("safe EXIF was not retained: %#v", metadata.SafeEXIF)
	}
	if _, ok := metadata.SafeEXIF["exif-ifd2-BodySerialNumber"]; ok {
		t.Fatal("camera serial number was retained")
	}
	if _, ok := metadata.SafeEXIF["exif-ifd3-GPSLatitude"]; ok {
		t.Fatal("GPS metadata was retained")
	}

	output, err := encodeColorManagedWebP(img, metadata, colorManagedWebPOptions{
		AlphaQuality: 100,
		Effort:       4,
		Lossless:     true,
		Quality:      100,
	})
	if err != nil {
		t.Fatalf("encodeColorManagedWebP() error = %v", err)
	}
	resultRef, err := vips.NewImageFromBuffer(output)
	if err != nil {
		t.Fatalf("load output webp: %v", err)
	}
	defer resultRef.Close()
	if !resultRef.HasICCProfile() {
		t.Fatal("output WebP does not contain an ICC profile")
	}
	if resultRef.Orientation() != 1 {
		t.Fatalf("output orientation = %d, want 1", resultRef.Orientation())
	}
	resultEXIF := resultRef.GetExif()
	if resultEXIF["exif-ifd0-Model"] == "" || resultEXIF["exif-ifd0-Copyright"] == "" {
		t.Fatalf("output safe EXIF is missing: %#v", resultEXIF)
	}
	for key := range resultEXIF {
		if key == "exif-ifd2-BodySerialNumber" || key == "exif-ifd3-GPSLatitude" {
			t.Fatalf("output contains sensitive EXIF %q", key)
		}
	}
}

func TestImageProcessorLibvipsSwitch(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, gradientImage(256, 256), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode source jpeg: %v", err)
	}
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_enabled", false)
	processor := NewImageProcessor(settings, "libvips-switch-secret", 10<<20, nil)

	legacy, err := processor.Process(context.Background(), ProcessImageInput{
		Data: source.Bytes(), Filename: "legacy.jpg", ContentType: "image/jpeg", Purpose: ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("legacy Process() error = %v", err)
	}
	legacyRef, err := vips.NewImageFromBuffer(legacy.Data)
	if err != nil {
		t.Fatalf("load legacy output: %v", err)
	}
	legacyHasICC := legacyRef.HasICCProfile()
	legacyRef.Close()

	_ = settings.Set(context.Background(), "image_libvips_enabled", true)
	managed, err := processor.Process(context.Background(), ProcessImageInput{
		Data: source.Bytes(), Filename: "managed.jpg", ContentType: "image/jpeg", Purpose: ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("managed Process() error = %v", err)
	}
	managedRef, err := vips.NewImageFromBuffer(managed.Data)
	if err != nil {
		t.Fatalf("load managed output: %v", err)
	}
	managedHasICC := managedRef.HasICCProfile()
	managedRef.Close()
	if legacyHasICC {
		t.Fatal("legacy encoder unexpectedly added an ICC profile")
	}
	if !managedHasICC {
		t.Fatal("libvips encoder did not add an sRGB ICC profile")
	}
}

func TestImageProcessorLibvipsKeepsLossyQualityAndWatermarkRoundTrip(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode source jpeg: %v", err)
	}
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "image_libvips_enabled", true)
	_ = settings.Set(context.Background(), "hidden_watermark_enabled", false)
	_ = settings.Set(context.Background(), "image_webp_lossless", false)
	_ = settings.Set(context.Background(), "image_webp_quality", 40)
	processor := NewImageProcessor(settings, "libvips-watermark-secret", 10<<20, nil)

	lowQuality, err := processor.Process(context.Background(), ProcessImageInput{
		Data: source.Bytes(), Filename: "quality-40.jpg", ContentType: "image/jpeg", Purpose: ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("quality 40 Process() error = %v", err)
	}
	_ = settings.Set(context.Background(), "image_webp_quality", 90)
	highQuality, err := processor.Process(context.Background(), ProcessImageInput{
		Data: source.Bytes(), Filename: "quality-90.jpg", ContentType: "image/jpeg", Purpose: ImagePurposeContent,
	})
	if err != nil {
		t.Fatalf("quality 90 Process() error = %v", err)
	}
	if lowQuality.ProcessedSize >= highQuality.ProcessedSize {
		t.Fatalf("lossy WebP quality setting is ineffective: q40=%d q90=%d", lowQuality.ProcessedSize, highQuality.ProcessedSize)
	}

	traceToken := "0102030405060708"
	processor.SetTraceResolver(func(_ context.Context, token string) bool { return token == traceToken })
	configureTestShortCodeResolver(processor, traceToken, testShortCode)
	protected, err := processor.Process(context.Background(), ProcessImageInput{
		Data:                  source.Bytes(),
		Filename:              "protected.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "viewer-account"},
		ForceWatermark:        true,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadToken: testShortCode,
	})
	if err != nil {
		t.Fatalf("protected Process() error = %v", err)
	}
	if !protected.WatermarkApplied {
		t.Fatalf("libvips watermark was not applied: %s", protected.WatermarkWarning)
	}
	extracted, err := processor.Extract(context.Background(), protected.Data)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !extracted.Found || !extracted.Valid || extracted.TraceToken != traceToken {
		t.Fatalf("libvips watermark round trip failed: %+v", extracted)
	}
}
