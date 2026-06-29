package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"sort"

	"github.com/disintegration/imaging"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
	"go.uber.org/zap"
)

func (p *ImageProcessor) Extract(ctx context.Context, data []byte) (HiddenWatermarkData, error) {
	return p.ExtractWithReference(ctx, data, nil)
}

func (p *ImageProcessor) VerifyMessagingTransport(ctx context.Context, data []byte, traceToken string, attempt int, preferRobust bool) error {
	return p.VerifyMessagingTransportWithReference(ctx, data, nil, traceToken, attempt, preferRobust)
}

func (p *ImageProcessor) VerifyMessagingTransportWithReference(ctx context.Context, data, referenceData []byte, traceToken string, attempt int, preferRobust bool) error {
	format, _, err := inspectImage(data)
	if err != nil {
		return err
	}
	img, err := decodeImage(data, format)
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	resized := imaging.Resize(
		img,
		max(1, int(math.Round(float64(bounds.Dx())*0.82))),
		max(1, int(math.Round(float64(bounds.Dy())*0.82))),
		imaging.Lanczos,
	)
	var resizedJPEG bytes.Buffer
	if err := jpeg.Encode(&resizedJPEG, resized, &jpeg.Options{Quality: 75}); err != nil {
		return err
	}
	if len(referenceData) == 0 {
		referenceData = data
	}
	if err := p.verifyTransportCandidate(ctx, resizedJPEG.Bytes(), referenceData, traceToken, attempt, preferRobust); err != nil {
		return fmt.Errorf("resize and jpeg recompression: %w", err)
	}
	return nil
}

func (p *ImageProcessor) verifyTransportCandidate(ctx context.Context, data, referenceData []byte, traceToken string, attempt int, preferRobust bool) error {
	input := ProcessImageInput{WatermarkAttempt: attempt, PreferRobust: preferRobust, ForceRemoteAdaptive: true}
	extracted, err := p.verifyExpectedTraceToken(ctx, data, referenceData, traceToken, "remote", input)
	if err != nil {
		return err
	}
	if !extracted.Found || !extracted.Valid || !extracted.TraceResolved || extracted.TraceToken != traceToken {
		return errors.New("watermark payload mismatch")
	}
	return nil
}

func (p *ImageProcessor) ExtractWithReference(ctx context.Context, data, referenceData []byte) (HiddenWatermarkData, error) {
	return p.ExtractWithReferenceProgress(ctx, data, referenceData, nil)
}

func (p *ImageProcessor) ExtractWithReferenceProgress(
	ctx context.Context,
	data, referenceData []byte,
	progress HiddenWatermarkProgressFunc,
) (HiddenWatermarkData, error) {
	if p == nil {
		return HiddenWatermarkData{}, errors.New("image processor is not configured")
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "decoding", Percent: 3})
	format, cfg, err := inspectImage(data)
	if err != nil {
		return HiddenWatermarkData{}, err
	}
	if format == "gif" || cfg.Width <= 0 || cfg.Height <= 0 {
		return HiddenWatermarkData{}, ErrImageUnsupported
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "decoding", Percent: 8})
	switch p.hiddenWatermarkEngine() {
	case "remote":
		result, err := p.extractRemoteHiddenWatermark(ctx, data, referenceData, progress)
		result.WatermarkEngine = "remote"
		return result, err
	case "auto":
		if p.remote != nil && p.remote.Configured() {
			remoteResult, remoteErr := p.extractRemoteHiddenWatermark(ctx, data, referenceData, progress)
			remoteResult.WatermarkEngine = "remote"
			if remoteErr == nil && remoteResult.Found {
				return remoteResult, nil
			}
			if remoteErr != nil {
				p.warn("remote hidden watermark extraction failed; falling back to local", zap.Error(remoteErr))
			}
		}
	}
	result, err := p.extractLocalHiddenWatermark(ctx, data, referenceData, format, progress)
	result.WatermarkEngine = "local"
	return result, err
}

func (p *ImageProcessor) extractLocalHiddenWatermark(
	ctx context.Context,
	data, referenceData []byte,
	format string,
	progress HiddenWatermarkProgressFunc,
) (HiddenWatermarkData, error) {
	img, err := decodeImage(data, format)
	if err != nil {
		return HiddenWatermarkData{}, ErrImageInvalid
	}
	candidates := make([]image.Image, 0, 10)
	if len(referenceData) > 0 {
		reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "recovering", Percent: 12})
		if referenceFormat, referenceConfig, inspectErr := inspectImage(referenceData); inspectErr == nil && referenceFormat != "gif" && referenceConfig.Width > 0 && referenceConfig.Height > 0 {
			if reference, decodeErr := decodeImage(referenceData, referenceFormat); decodeErr == nil {
				candidates = append(candidates, localHiddenWatermarkRecoveryCandidates(img, reference)...)
			}
		}
		reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "recovering", Percent: 25})
	}
	direct, _, _ := hiddenWatermarkWorkingImage(img)
	candidates = append(candidates, direct)
	specs := p.hiddenWatermarkExtractSpecs()
	profiles := p.hiddenWatermarkExtractProfiles()
	total := max(1, len(candidates)*len(specs)*len(profiles))
	completed := 0
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
		Stage: "extracting", Percent: 30, Completed: completed, Total: total,
	})
	extractCtx, cancel := context.WithTimeout(ctx, hiddenWatermarkTimeout)
	defer cancel()
	var lastErr error
	for _, candidate := range candidates {
		for _, spec := range specs {
			for _, profile := range profiles {
				decoder, extractErr := watermark.Extract(
					extractCtx,
					candidate,
					mark.NewExtract(spec.encodedSize*8, markOptionsForHiddenWatermarkProfile(profile)...),
					watermarkOptionsForHiddenWatermarkProfile(profile)...,
				)
				completed++
				reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{
					Stage:     "extracting",
					Percent:   30 + (completed * 64 / total),
					Completed: completed,
					Total:     total,
				})
				if extractErr != nil {
					if !errors.Is(extractErr, watermark.ErrTooSmallImage) {
						lastErr = extractErr
					}
					continue
				}
				result := p.decodeExtractedWatermarkPayload(ctx, decoder.DecodeToBytes(), spec.payloadSize)
				if result.Found {
					reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "verifying", Percent: 98, Completed: completed, Total: total})
					return result, nil
				}
			}
		}
	}
	if lastErr != nil {
		return HiddenWatermarkData{}, lastErr
	}
	reportHiddenWatermarkProgress(progress, HiddenWatermarkProgress{Stage: "verifying", Percent: 98, Completed: completed, Total: total})
	return HiddenWatermarkData{Found: false}, nil
}

const (
	localRecoveryAspectDelta = 0.08
	localRecoveryMinArea     = 0.35
	localRecoveryMaxImages   = 8
)

func localHiddenWatermarkRecoveryCandidates(screenshot, reference image.Image) []image.Image {
	target, _, _ := hiddenWatermarkWorkingImage(reference)
	targetBounds := target.Bounds()
	if targetBounds.Dx() <= 0 || targetBounds.Dy() <= 0 {
		return nil
	}

	rotations := []image.Image{
		screenshot,
		imaging.Rotate90(screenshot),
		imaging.Rotate270(screenshot),
		imaging.Rotate180(screenshot),
	}
	candidates := make([]image.Image, 0, localRecoveryMaxImages)
	for _, rotated := range rotations {
		if crop, ok := localWatermarkContentCrop(rotated); ok {
			for _, rect := range localWatermarkCropRectVariants(crop, rotated.Bounds()) {
				appendLocalWatermarkAlignedCandidate(&candidates, imaging.Crop(rotated, rect), targetBounds)
				if len(candidates) >= localRecoveryMaxImages {
					return candidates
				}
			}
		}
		if localWatermarkAspectDelta(rotated.Bounds(), targetBounds) <= localRecoveryAspectDelta && localWatermarkAreaRatio(rotated.Bounds(), targetBounds) >= localRecoveryMinArea {
			appendLocalWatermarkAlignedCandidate(&candidates, rotated, targetBounds)
			if len(candidates) >= localRecoveryMaxImages {
				return candidates
			}
		}
	}
	return candidates
}

func appendLocalWatermarkAlignedCandidate(candidates *[]image.Image, candidate image.Image, target image.Rectangle) {
	if candidate == nil || candidate.Bounds().Dx() <= 0 || candidate.Bounds().Dy() <= 0 {
		return
	}
	aligned := candidate
	if candidate.Bounds().Dx() != target.Dx() || candidate.Bounds().Dy() != target.Dy() {
		aligned = imaging.Resize(candidate, target.Dx(), target.Dy(), imaging.Lanczos)
	}
	aligned, _, _ = hiddenWatermarkWorkingImage(aligned)
	for _, existing := range *candidates {
		if localWatermarkImageFingerprint(existing) == localWatermarkImageFingerprint(aligned) {
			return
		}
	}
	*candidates = append(*candidates, aligned)
}

func localWatermarkContentCrop(img image.Image) (image.Rectangle, bool) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width < 64 || height < 64 {
		return image.Rectangle{}, false
	}
	step := max(1, min(width, height)/160)
	red := make([]int, 0, (width+height)*2/step+4)
	green := make([]int, 0, (width+height)*2/step+4)
	blue := make([]int, 0, (width+height)*2/step+4)
	appendSample := func(x, y int) {
		r, g, b, _ := img.At(x, y).RGBA()
		red = append(red, int(r>>8))
		green = append(green, int(g>>8))
		blue = append(blue, int(b>>8))
	}
	for x := bounds.Min.X; x < bounds.Max.X; x += step {
		appendSample(x, bounds.Min.Y)
		appendSample(x, bounds.Max.Y-1)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		appendSample(bounds.Min.X, y)
		appendSample(bounds.Max.X-1, y)
	}
	sort.Ints(red)
	sort.Ints(green)
	sort.Ints(blue)
	background := [3]int{red[len(red)/2], green[len(green)/2], blue[len(blue)/2]}
	nearBorder := 0
	for index := range red {
		if localWatermarkColorDistanceSquared(red[index], green[index], blue[index], background) <= 18*18 {
			nearBorder++
		}
	}
	if float64(nearBorder)/float64(len(red)) < 0.45 {
		return image.Rectangle{}, false
	}

	activeColumns := make([]int, 0, width)
	yStep := max(1, height/384)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		active, samples := 0, 0
		for y := bounds.Min.Y; y < bounds.Max.Y; y += yStep {
			r, g, b, _ := img.At(x, y).RGBA()
			if localWatermarkColorDistanceSquared(int(r>>8), int(g>>8), int(b>>8), background) > 26*26 {
				active++
			}
			samples++
		}
		if samples > 0 && float64(active)/float64(samples) > 0.45 {
			activeColumns = append(activeColumns, x)
		}
	}
	x1, x2, ok := localWatermarkLongestRun(activeColumns)
	if !ok {
		return image.Rectangle{}, false
	}

	activeRows := make([]int, 0, height)
	xStep := max(1, (x2-x1)/384)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		active, samples := 0, 0
		for x := x1; x < x2; x += xStep {
			r, g, b, _ := img.At(x, y).RGBA()
			if localWatermarkColorDistanceSquared(int(r>>8), int(g>>8), int(b>>8), background) > 26*26 {
				active++
			}
			samples++
		}
		if samples > 0 && float64(active)/float64(samples) > 0.75 {
			activeRows = append(activeRows, y)
		}
	}
	y1, y2, ok := localWatermarkLongestRun(activeRows)
	if !ok {
		return image.Rectangle{}, false
	}
	crop := image.Rect(x1, y1, x2, y2).Intersect(bounds)
	areaRatio := float64(crop.Dx()*crop.Dy()) / float64(width*height)
	return crop, areaRatio >= 0.15 && areaRatio <= 0.98
}

func localWatermarkCropRectVariants(rect, bounds image.Rectangle) []image.Rectangle {
	variants := []image.Rectangle{rect}
	for _, delta := range []int{-1, 1} {
		candidate := image.Rect(rect.Min.X-delta, rect.Min.Y-delta, rect.Max.X+delta, rect.Max.Y+delta).Intersect(bounds)
		if candidate.Dx() > 0 && candidate.Dy() > 0 {
			variants = append(variants, candidate)
		}
	}
	return variants
}

func localWatermarkLongestRun(values []int) (int, int, bool) {
	if len(values) == 0 {
		return 0, 0, false
	}
	bestStart, bestEnd := values[0], values[0]+1
	start, previous := values[0], values[0]
	for _, value := range values[1:] {
		if value == previous+1 {
			previous = value
			continue
		}
		if previous+1-start > bestEnd-bestStart {
			bestStart, bestEnd = start, previous+1
		}
		start, previous = value, value
	}
	if previous+1-start > bestEnd-bestStart {
		bestStart, bestEnd = start, previous+1
	}
	return bestStart, bestEnd, true
}

func localWatermarkColorDistanceSquared(red, green, blue int, background [3]int) int {
	dr, dg, db := red-background[0], green-background[1], blue-background[2]
	return dr*dr + dg*dg + db*db
}

func localWatermarkAspectDelta(first, second image.Rectangle) float64 {
	if first.Dx() <= 0 || first.Dy() <= 0 || second.Dx() <= 0 || second.Dy() <= 0 {
		return math.Inf(1)
	}
	return math.Abs(math.Log((float64(first.Dx()) / float64(first.Dy())) / (float64(second.Dx()) / float64(second.Dy()))))
}

func localWatermarkAreaRatio(first, second image.Rectangle) float64 {
	if second.Dx() <= 0 || second.Dy() <= 0 {
		return 0
	}
	return float64(first.Dx()*first.Dy()) / float64(second.Dx()*second.Dy())
}

func localWatermarkImageFingerprint(img image.Image) string {
	bounds := img.Bounds()
	points := [][2]int{{0, 0}, {1, 1}, {2, 3}, {3, 2}}
	values := make([]byte, 0, 12)
	for _, point := range points {
		x := bounds.Min.X + point[0]*max(bounds.Dx()-1, 0)/3
		y := bounds.Min.Y + point[1]*max(bounds.Dy()-1, 0)/3
		r, g, b, _ := img.At(x, y).RGBA()
		values = append(values, byte(r>>8), byte(g>>8), byte(b>>8))
	}
	return fmt.Sprintf("%dx%d:%x", bounds.Dx(), bounds.Dy(), values)
}
