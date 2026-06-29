package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"mime"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	avif "github.com/gen2brain/avif"
	webp "github.com/gen2brain/webp"
	"github.com/yyyoichi/watermark_zero/mark"
	xwebp "golang.org/x/image/webp"
)

func fitImageMaxDimension(img image.Image, maxDimension int) image.Image {
	if img == nil || maxDimension <= 0 {
		return img
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	longest := max(width, height)
	if longest <= maxDimension {
		return img
	}
	scale := float64(maxDimension) / float64(longest)
	return imaging.Resize(
		img,
		max(1, int(math.Round(float64(width)*scale))),
		max(1, int(math.Round(float64(height)*scale))),
		imaging.Lanczos,
	)
}

func fitImageMinDimension(img image.Image, minDimension int) image.Image {
	if img == nil || minDimension <= 0 {
		return img
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	longest := max(width, height)
	if longest >= minDimension || longest <= 0 {
		return img
	}
	scale := float64(minDimension) / float64(longest)
	return imaging.Resize(
		img,
		max(1, int(math.Round(float64(width)*scale))),
		max(1, int(math.Round(float64(height)*scale))),
		imaging.Lanczos,
	)
}

func hiddenWatermarkWorkingImage(img image.Image) (image.Image, image.Rectangle, bool) {
	if img == nil {
		return img, image.Rectangle{}, false
	}
	bounds := img.Bounds()
	area := int64(bounds.Dx()) * int64(bounds.Dy())
	if area <= maxWatermarkPixels {
		return img, bounds, false
	}
	scale := math.Sqrt(float64(maxWatermarkPixels) / float64(area))
	width := max(1, int(math.Floor(float64(bounds.Dx())*scale)))
	height := max(1, int(math.Floor(float64(bounds.Dy())*scale)))
	left := bounds.Min.X + (bounds.Dx()-width)/2
	top := bounds.Min.Y + (bounds.Dy()-height)/2
	rect := image.Rect(left, top, left+width, top+height)
	return imaging.Crop(img, rect), rect, true
}

func hiddenWatermarkCapacity(bounds image.Rectangle, profile hiddenWatermarkProfile) int {
	if profile.blockWidth <= 0 || profile.blockHeight <= 0 {
		return 0
	}
	return ((bounds.Dx() + 1) / profile.blockWidth) * ((bounds.Dy() + 1) / profile.blockHeight)
}

func hiddenWatermarkEncodedBits(payloadSize int) int {
	return mark.NewExtract(payloadSize*8, mark.WithGolay(mark.DefaultShuffleSeed)).Len()
}

func (p *ImageProcessor) resize(img image.Image) image.Image {
	maxWidth := p.intSetting("image_max_width", 0, 0, maxDecodedImageDimension)
	maxHeight := p.intSetting("image_max_height", 0, 0, maxDecodedImageDimension)
	if maxWidth == 0 && maxHeight == 0 {
		return img
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if maxWidth == 0 {
		maxWidth = width
	}
	if maxHeight == 0 {
		maxHeight = height
	}
	if width <= maxWidth && height <= maxHeight {
		return img
	}
	return imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)
}

func (p *ImageProcessor) encode(img image.Image, sourceFormat string, purpose ImagePurpose, forceWebP, forceLossless, protected bool, useLibvips bool, colorMetadata imageColorMetadata) ([]byte, string, string, error) {
	quality := p.intSetting("image_webp_quality", 85, 1, 100)
	if protected {
		quality = p.intSetting("image_protection_webp_quality", 95, 1, 100)
	}
	if purpose == ImagePurposeAvatar {
		quality = p.intSetting("image_avatar_webp_quality", 75, 1, 100)
	}
	if forceWebP || p.boolSetting("image_webp_enabled", true) || sourceFormat == "webp" {
		alphaQuality := p.intSetting("image_webp_alpha_quality", 100, 0, 100)
		lossless := forceLossless || p.boolSetting("image_webp_lossless", false)
		method := p.intSetting("image_webp_method", 4, 0, 6)
		if useLibvips {
			data, err := encodeColorManagedWebP(img, colorMetadata, colorManagedWebPOptions{
				AlphaQuality: alphaQuality,
				Effort:       method,
				Lossless:     lossless,
				Quality:      quality,
			})
			return data, "webp", "image/webp", err
		}
		if alphaQuality < 100 {
			img = quantizeImageAlpha(img, alphaQuality)
		}
		var buf bytes.Buffer
		err := webp.Encode(&buf, img, webp.Options{
			Quality:  quality,
			Lossless: lossless,
			Method:   method,
			Exact:    alphaQuality == 100,
		})
		return buf.Bytes(), "webp", "image/webp", err
	}

	var buf bytes.Buffer
	switch sourceFormat {
	case "jpeg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		return buf.Bytes(), "jpeg", "image/jpeg", err
	case "png":
		err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img)
		return buf.Bytes(), "png", "image/png", err
	default:
		return nil, "", "", ErrImageUnsupported
	}
}

func inspectImage(data []byte) (string, image.Config, error) {
	if len(data) < 12 {
		return "", image.Config{}, ErrImageInvalid
	}
	if isAnimatedAVIF(data) {
		return "", image.Config{}, ErrImageAnimatedUnsupported
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", image.Config{}, ErrImageInvalid
	}
	format = strings.ToLower(format)
	switch format {
	case "jpeg":
	case "png":
		if bytes.Contains(data, []byte("acTL")) {
			return "", image.Config{}, ErrImageAnimatedUnsupported
		}
	case "webp":
		if bytes.Contains(data, []byte("ANIM")) || bytes.Contains(data, []byte("ANMF")) {
			return "", image.Config{}, ErrImageAnimatedUnsupported
		}
	case "avif":
	case "gif":
		return "", image.Config{}, ErrImageAnimatedUnsupported
	default:
		return "", image.Config{}, ErrImageUnsupported
	}
	return format, cfg, nil
}

func decodeImage(data []byte, format string) (image.Image, error) {
	if format == "avif" {
		img, err := avif.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, ErrImageInvalid
		}
		return img, nil
	}
	if format == "webp" {
		img, err := xwebp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, ErrImageInvalid
		}
		return img, nil
	}
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrImageInvalid
	}
	return img, nil
}

func validateDeclaredImageMIME(declared, format string) error {
	declared, _, _ = mime.ParseMediaType(strings.TrimSpace(declared))
	if declared == "" || declared == "application/octet-stream" {
		return nil
	}
	expected := map[string]string{
		"avif": "image/avif",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
	}[format]
	if format == "jpeg" && strings.EqualFold(declared, "image/jpg") {
		return nil
	}
	if expected == "" || !strings.EqualFold(declared, expected) {
		return ErrImageMIME
	}
	return nil
}

func isAnimatedAVIF(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	boxSize := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if boxSize == 0 {
		boxSize = len(data)
	}
	if boxSize < 12 || boxSize > len(data) || !bytes.Equal(data[4:8], []byte("ftyp")) {
		return false
	}
	for offset := 8; offset+4 <= boxSize; offset += 4 {
		if offset == 12 {
			continue
		}
		if bytes.Equal(data[offset:offset+4], []byte("avis")) {
			return true
		}
	}
	return false
}

func processedImageFilename(filename, format string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if strings.TrimSpace(base) == "" {
		base = "image"
	}
	ext := "." + format
	if format == "jpeg" {
		ext = ".jpg"
	}
	return base + ext
}

func quantizeImageAlpha(src image.Image, quality int) image.Image {
	if quality >= 100 {
		return src
	}
	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	step := uint8(1 + (100-quality)/5)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			if step > 1 {
				quantized := min(((int(nrgba.A)+int(step)/2)/int(step))*int(step), 255)
				nrgba.A = uint8(quantized)
			}
			dst.SetNRGBA(x, y, nrgba)
		}
	}
	return dst
}

func firstNonEmptyImageString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
