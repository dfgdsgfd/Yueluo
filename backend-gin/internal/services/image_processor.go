package services

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/yyyoichi/watermark_zero/mark"
	"go.uber.org/zap"

	"yuem-go/backend-gin/internal/domain"
)

type ImagePurpose string

const (
	ImagePurposeContent    ImagePurpose = "content"
	ImagePurposeAvatar     ImagePurpose = "avatar"
	ImagePurposeBackground ImagePurpose = "background"
	ImagePurposeCover      ImagePurpose = "cover"
	ImagePurposeFeedback   ImagePurpose = "feedback"
	ImagePurposeAIAnalysis ImagePurpose = "ai_analysis"
)

const (
	hiddenTraceTokenSize             = domain.ImageWatermarkTraceTokenBytes
	hiddenWatermarkMinSpatialRepeats = 5
	hiddenWatermarkTimeout           = 15 * time.Second
	protectedWatermarkMinDimension   = 720
	protectedWatermarkMaxDimension   = 1280
	hiddenWatermarkBlockWidth        = 8
	hiddenWatermarkBlockHeight       = 6
	hiddenWatermarkD1                = 21
	hiddenWatermarkD2                = 9
	maxDecodedImagePixels            = 50_000_000
	maxDecodedImageDimension         = 16_384
	maxWatermarkPixels               = 4_000_000
)

var (
	ErrImageInvalid             = errors.New("error.image_invalid")
	ErrImageUnsupported         = errors.New("error.image_unsupported")
	ErrImageAnimatedUnsupported = errors.New("error.image_animated_unsupported")
	ErrImageMIME                = errors.New("error.image_mime_mismatch")
	ErrImageTooLarge            = errors.New("error.image_too_large")
)

type ImageProcessor struct {
	settings          *SettingsService
	secret            []byte
	maxBytes          int64
	logger            *zap.Logger
	gate              imageProcessingGate
	remote            *HiddenWatermarkRemoteClient
	requireRemote     bool
	traceResolver     func(context.Context, string) bool
	payloadResolver   func(context.Context, []byte) HiddenWatermarkData
	dimensionResolver func(context.Context) [][2]int
}

type ProcessImageInput struct {
	Data                  []byte
	Filename              string
	ContentType           string
	Purpose               ImagePurpose
	User                  *RequestUser
	ForceWatermark        bool
	WatermarkTraceToken   string
	WatermarkPayloadToken string
	WatermarkAttempt      int
	MinDimension          int
	MaxDimension          int
	PreferRobust          bool
	ForceRemoteAdaptive   bool
	VerifyWithReference   bool
	WatermarkProgress     HiddenWatermarkProgressFunc
}

type hiddenWatermarkProfile struct {
	blockWidth      int
	blockHeight     int
	coefficientMode string
	d1              int
	d2              int
	eccMode         string
	golaySeed       int64
}

type hiddenWatermarkRemoteProfile struct {
	name        string
	passwordWM  int
	passwordImg int
	d1          int
	d2          int
}

var hiddenWatermarkProfiles = [...]hiddenWatermarkProfile{
	{blockWidth: 8, blockHeight: 6, coefficientMode: "d1d2", d1: 21, d2: 9, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed},
	{blockWidth: 6, blockHeight: 4, coefficientMode: "d1d2", d1: 36, d2: 20, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed},
	{blockWidth: 8, blockHeight: 6, coefficientMode: "d1d2", d1: 10, d2: 4, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed},
	{blockWidth: 6, blockHeight: 4, coefficientMode: "d1d2", d1: 16, d2: 7, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed},
}

var officialHiddenWatermarkRemoteProfile = hiddenWatermarkRemoteProfile{name: "official", passwordWM: 1, passwordImg: 1, d1: 36, d2: 20}

var hiddenWatermarkRemoteStrengthProfiles = [...]hiddenWatermarkRemoteProfile{
	{name: "fidelity", passwordWM: 1, passwordImg: 1, d1: 18, d2: 8},
	{name: "balanced", passwordWM: 1, passwordImg: 1, d1: 24, d2: 12},
	{name: "strong", passwordWM: 1, passwordImg: 1, d1: 30, d2: 15},
	officialHiddenWatermarkRemoteProfile,
}

type ProcessedImage struct {
	Data                  []byte `json:"-"`
	Filename              string `json:"filename"`
	ContentType           string `json:"contentType"`
	Format                string `json:"format"`
	OriginalSize          int64  `json:"originalSize"`
	ProcessedSize         int64  `json:"processedSize"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	WatermarkApplied      bool   `json:"watermarkApplied"`
	WatermarkWarning      string `json:"watermarkWarning,omitempty"`
	WatermarkTraceToken   string `json:"watermarkTraceToken,omitempty"`
	WatermarkPayloadBytes int    `json:"watermarkPayloadBytes,omitempty"`
	WatermarkEngine       string `json:"watermarkEngine,omitempty"`
	WatermarkReference    []byte `json:"-"`
}

type HiddenWatermarkData struct {
	Found           bool      `json:"found"`
	Valid           bool      `json:"valid"`
	Version         int       `json:"version,omitempty"`
	UID             int64     `json:"uid,omitempty"`
	UserID          string    `json:"userId,omitempty"`
	Username        string    `json:"username,omitempty"`
	UploadedAt      time.Time `json:"uploadedAt"`
	SourceHash      string    `json:"sourceHash,omitempty"`
	CustomText      string    `json:"customText,omitempty"`
	PostID          int64     `json:"postId,omitempty"`
	ImageID         int64     `json:"imageId,omitempty"`
	JobID           string    `json:"jobId,omitempty"`
	TraceToken      string    `json:"traceToken,omitempty"`
	TraceType       string    `json:"traceType,omitempty"`
	TraceResolved   bool      `json:"traceResolved"`
	PayloadBytes    int       `json:"payloadBytes,omitempty"`
	PayloadBits     int       `json:"payloadBits,omitempty"`
	PayloadFormat   string    `json:"payloadFormat,omitempty"`
	WatermarkEngine string    `json:"watermarkEngine,omitempty"`
	IncludedFields  []string  `json:"includedFields,omitempty"`
}

type hiddenWatermarkExtractSpec struct {
	payloadSize int
	encodedSize int
}

type imageProcessingGate struct {
	mu     sync.Mutex
	active int
}

func NewImageProcessor(settings *SettingsService, secret string, maxBytes int64, logger *zap.Logger) *ImageProcessor {
	return &ImageProcessor{
		settings: settings,
		secret:   []byte(secret),
		maxBytes: maxBytes,
		logger:   logger,
	}
}

func NewImageProcessorWithRemote(settings *SettingsService, secret string, maxBytes int64, logger *zap.Logger, remote HiddenWatermarkRemoteClientConfig) *ImageProcessor {
	processor := NewImageProcessor(settings, secret, maxBytes, logger)
	processor.remote = NewHiddenWatermarkRemoteClient(remote)
	return processor
}

func (p *ImageProcessor) SetTraceResolver(resolver func(context.Context, string) bool) {
	if p != nil {
		p.traceResolver = resolver
	}
}

func (p *ImageProcessor) SetPayloadResolver(resolver func(context.Context, []byte) HiddenWatermarkData) {
	if p != nil {
		p.payloadResolver = resolver
	}
}

func (p *ImageProcessor) SetRecoverDimensionResolver(resolver func(context.Context) [][2]int) {
	if p != nil {
		p.dimensionResolver = resolver
	}
}

func (p *ImageProcessor) RequireRemoteHiddenWatermark() {
	if p != nil {
		p.requireRemote = true
	}
}

func NormalizeImagePurpose(value string) ImagePurpose {
	switch ImagePurpose(strings.ToLower(strings.TrimSpace(value))) {
	case ImagePurposeAvatar:
		return ImagePurposeAvatar
	case ImagePurposeBackground:
		return ImagePurposeBackground
	case ImagePurposeCover:
		return ImagePurposeCover
	case ImagePurposeFeedback:
		return ImagePurposeFeedback
	case ImagePurposeAIAnalysis:
		return ImagePurposeAIAnalysis
	default:
		return ImagePurposeContent
	}
}

func (p *ImageProcessor) Process(ctx context.Context, input ProcessImageInput) (ProcessedImage, error) {
	if p == nil {
		return ProcessedImage{}, errors.New("image processor is not configured")
	}
	if len(input.Data) == 0 {
		return ProcessedImage{}, ErrImageInvalid
	}
	if p.maxBytes > 0 && int64(len(input.Data)) > p.maxBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if err := p.gate.acquire(ctx, p.intSetting("image_processing_concurrency", 1, 1, 8)); err != nil {
		return ProcessedImage{}, err
	}
	defer p.gate.release()

	format, cfg, err := inspectImage(input.Data)
	if err != nil {
		return ProcessedImage{}, err
	}
	if err := validateDeclaredImageMIME(input.ContentType, format); err != nil {
		return ProcessedImage{}, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		cfg.Width > maxDecodedImageDimension || cfg.Height > maxDecodedImageDimension ||
		int64(cfg.Width)*int64(cfg.Height) > maxDecodedImagePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	watermarkAllowed := input.Purpose != ImagePurposeAvatar && input.Purpose != ImagePurposeBackground
	watermarkEnabled := watermarkAllowed && (input.ForceWatermark ||
		(p.boolSetting("hidden_watermark_enabled", true) && !p.boolSetting("hidden_watermark_protected_only", true)))
	useLibvips := p.boolSetting("image_libvips_enabled", false)
	colorMetadata := imageColorMetadata{}
	var img image.Image
	if useLibvips {
		img, colorMetadata, err = decodeWithColorManagement(input.Data)
		if err != nil {
			p.warn("libvips color processing unavailable; using legacy image decoder", zap.Error(err))
			useLibvips = false
		}
	}
	if !useLibvips {
		img, err = decodeImage(input.Data, format)
	}
	if err != nil {
		return ProcessedImage{}, ErrImageInvalid
	}
	img = p.resize(img)
	if input.MinDimension > 0 {
		img = fitImageMinDimension(img, min(input.MinDimension, maxDecodedImageDimension))
	}
	if input.MaxDimension > 0 {
		img = fitImageMaxDimension(img, min(input.MaxDimension, maxDecodedImageDimension))
	}
	profiles := p.hiddenWatermarkProfilesForInput(input)
	embeddingPayloadSize := hiddenWatermarkEmbeddingPayloadSize(input)
	if watermarkEnabled && p.useRemoteHiddenWatermark() {
		remoteCapacityProfile := hiddenWatermarkProfile{blockWidth: 8, blockHeight: 8}
		img, err = ensureHiddenWatermarkCapacity(img, remoteCapacityProfile, embeddingPayloadSize*8*hiddenWatermarkMinSpatialRepeats)
		if err != nil {
			return ProcessedImage{}, err
		}
	}
	if watermarkEnabled && p.hiddenWatermarkEngine() != "remote" {
		profileIndex := min(max(input.WatermarkAttempt, 0), len(profiles)-1)
		img, err = ensureHiddenWatermarkCapacity(img, profiles[profileIndex], hiddenWatermarkEncodedBits(embeddingPayloadSize)*hiddenWatermarkMinSpatialRepeats)
		if err != nil {
			return ProcessedImage{}, err
		}
	}

	result := ProcessedImage{
		OriginalSize: int64(len(input.Data)),
		Width:        img.Bounds().Dx(),
		Height:       img.Bounds().Dy(),
	}
	var verificationReference []byte
	webpEnabled := input.ForceWatermark || p.boolSetting("image_webp_enabled", true)
	if watermarkEnabled {
		marked, traceToken, watermarkEngine, watermarkErr := p.embedHiddenWatermark(ctx, img, input)
		if watermarkErr != nil {
			result.WatermarkWarning = watermarkErr.Error()
			p.warn("hidden watermark embedding failed",
				zap.Error(watermarkErr),
				zap.String("purpose", string(input.Purpose)),
				zap.Int("width", result.Width),
				zap.Int("height", result.Height),
			)
			if input.ForceWatermark {
				return ProcessedImage{}, watermarkErr
			}
		} else {
			img = marked
			if input.VerifyWithReference && watermarkEngine == "remote" {
				verificationReference, _ = encodePNGImage(marked)
			}
			result.WatermarkApplied = true
			result.WatermarkTraceToken = traceToken
			result.WatermarkPayloadBytes = embeddingPayloadSize
			result.WatermarkEngine = watermarkEngine
			result.WatermarkReference = verificationReference
		}
	}

	forceWebP := result.WatermarkApplied && webpEnabled
	forceLossless := forceWebP
	if input.ForceWatermark && p.stringSetting("image_protection_output_mode") == "quality_webp" {
		forceLossless = false
	}
	encoded, outputFormat, contentType, err := p.encode(img, format, input.Purpose, forceWebP, forceLossless, input.ForceWatermark, useLibvips, colorMetadata)
	if err != nil {
		return ProcessedImage{}, fmt.Errorf("encode processed image: %w", err)
	}
	result.Data = encoded
	result.Format = outputFormat
	result.ContentType = contentType
	result.ProcessedSize = int64(len(encoded))
	result.Filename = processedImageFilename(input.Filename, outputFormat)
	if result.WatermarkApplied {
		reportHiddenWatermarkProgress(input.WatermarkProgress, HiddenWatermarkProgress{
			Stage: "direct_verification", Percent: 96,
		})
		extracted, extractErr := p.verifyExpectedTraceToken(ctx, encoded, verificationReference, result.WatermarkTraceToken, result.WatermarkEngine, input)
		if extractErr != nil || !extracted.Found || !extracted.Valid || extracted.TraceToken != result.WatermarkTraceToken {
			result.WatermarkApplied = false
			result.WatermarkWarning = "hidden watermark self-verification failed"
			fields := []zap.Field{
				zap.String("purpose", string(input.Purpose)),
				zap.Bool("found", extracted.Found),
				zap.Bool("valid", extracted.Valid),
			}
			if extractErr != nil {
				fields = append(fields, zap.Error(extractErr))
			}
			p.warn("hidden watermark self-verification failed", fields...)
			if input.ForceWatermark {
				if extractErr != nil {
					return ProcessedImage{}, fmt.Errorf("hidden watermark self-verification failed: %w", extractErr)
				}
				return ProcessedImage{}, errors.New("hidden watermark self-verification failed")
			}
		}
	}
	return result, nil
}
