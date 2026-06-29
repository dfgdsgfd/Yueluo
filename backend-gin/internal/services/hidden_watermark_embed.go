package services

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/disintegration/imaging"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
	"go.uber.org/zap"

	"yuem-go/backend-gin/internal/domain"
)

func (p *ImageProcessor) embedHiddenWatermark(ctx context.Context, img image.Image, input ProcessImageInput) (image.Image, string, string, error) {
	encoded, traceToken, err := buildWatermarkPayload(input.WatermarkTraceToken, input.WatermarkPayloadToken)
	if err != nil {
		return nil, "", "", err
	}
	if p.requireRemote {
		marked, err := p.embedRemoteHiddenWatermark(ctx, img, encoded, input)
		return marked, traceToken, "remote", err
	}
	switch p.hiddenWatermarkEngine() {
	case "remote":
		marked, err := p.embedRemoteHiddenWatermark(ctx, img, encoded, input)
		return marked, traceToken, "remote", err
	case "auto":
		if p.remote != nil && p.remote.Configured() {
			if marked, remoteErr := p.embedRemoteHiddenWatermark(ctx, img, encoded, input); remoteErr == nil {
				return marked, traceToken, "remote", nil
			} else {
				p.warn("remote hidden watermark embedding failed; falling back to local", zap.Error(remoteErr))
			}
		}
		marked, localErr := p.embedLocalHiddenWatermark(ctx, img, input, encoded)
		return marked, traceToken, "local", localErr
	default:
		marked, err := p.embedLocalHiddenWatermark(ctx, img, input, encoded)
		return marked, traceToken, "local", err
	}
}

func buildWatermarkPayload(traceTokenValue, payloadTokenValue string) ([]byte, string, error) {
	traceToken := strings.ToLower(strings.TrimSpace(traceTokenValue))
	payloadToken := strings.ToLower(strings.TrimSpace(payloadTokenValue))
	if traceToken == "" {
		return nil, "", errors.New("image watermark trace token is required")
	}
	traceDecoded, err := hex.DecodeString(traceToken)
	if err != nil || len(traceDecoded) != hiddenTraceTokenSize {
		return nil, "", errors.New("image watermark trace token is invalid")
	}
	if payloadToken == "" {
		return nil, "", errors.New("image watermark payload token is required")
	}
	payloadDecoded, err := hex.DecodeString(payloadToken)
	if err != nil || len(payloadDecoded) < domain.ImageWatermarkMinShortCodeBytes || len(payloadDecoded) > domain.ImageWatermarkShortCodeBytes {
		return nil, "", errors.New("image watermark payload token is invalid")
	}
	return payloadDecoded, traceToken, nil
}

func (p *ImageProcessor) embedLocalHiddenWatermark(ctx context.Context, img image.Image, input ProcessImageInput, encoded []byte) (image.Image, error) {
	embedCtx, cancel := context.WithTimeout(ctx, hiddenWatermarkTimeout)
	defer cancel()
	profiles := p.hiddenWatermarkProfilesForInput(input)
	profileIndex := min(max(input.WatermarkAttempt, 0), len(profiles)-1)
	profile := profiles[profileIndex]
	workingImage, workingRect, cropped := hiddenWatermarkWorkingImage(img)
	marked, err := watermark.Embed(
		embedCtx,
		workingImage,
		mark.NewBytes(encoded, markOptionsForHiddenWatermarkProfile(profile)...),
		watermarkOptionsForHiddenWatermarkProfile(profile)...,
	)
	if err != nil || !cropped {
		return marked, err
	}
	return imaging.Paste(imaging.Clone(img), marked, workingRect.Min.Sub(img.Bounds().Min)), nil
}

func (p *ImageProcessor) embedRemoteHiddenWatermark(ctx context.Context, img image.Image, encoded []byte, input ProcessImageInput) (image.Image, error) {
	if p.remote == nil || !p.remote.Configured() {
		return nil, fmt.Errorf("%w: hidden watermark remote service is not configured", ErrHiddenWatermarkRemoteUnavailable)
	}
	remoteCtx, cancel := p.remoteWatermarkRequestContext(ctx)
	defer cancel()
	imageData, err := encodePNGImage(img)
	if err != nil {
		return nil, err
	}
	options := p.remoteWatermarkOptionsForImage(img, input)
	var profileProgress HiddenWatermarkProgressFunc
	if input.WatermarkProgress != nil {
		profileProgress = func(value HiddenWatermarkProgress) {
			value.Profile = options.Profile
			reportHiddenWatermarkProgress(input.WatermarkProgress, value)
		}
	}
	markedData, err := p.remote.EmbedWithProgress(remoteCtx, imageData, encoded, options, profileProgress)
	if err != nil {
		return nil, err
	}
	format, _, err := inspectImage(markedData)
	if err != nil {
		return nil, err
	}
	marked, err := decodeImage(markedData, format)
	if err != nil {
		return nil, err
	}
	return marked, nil
}

func (p *ImageProcessor) extractRemoteHiddenWatermark(
	ctx context.Context,
	data, referenceData []byte,
	progress HiddenWatermarkProgressFunc,
) (HiddenWatermarkData, error) {
	if p.remote == nil || !p.remote.Configured() {
		return HiddenWatermarkData{}, errors.New("hidden watermark remote service is not configured")
	}
	remoteCtx, cancel := p.remoteWatermarkRequestContext(ctx)
	defer cancel()
	specs := p.hiddenWatermarkExtractSpecs()
	payloadSizes := hiddenWatermarkEncodedPayloadSizes(specs)
	recoverDimensions := p.remoteWatermarkRecoverDimensions(ctx, data, referenceData)
	profiles := p.remoteWatermarkExtractProfiles()
	var lastErr error
	succeeded := false
	for profileIndex, profile := range profiles {
		options := p.remoteWatermarkOptionsForProfile(profile)
		options.RecoverDimensions = recoverDimensions
		profileProgress := func(value HiddenWatermarkProgress) {
			value.Percent = (profileIndex*100 + value.Percent) / max(1, len(profiles))
			value.Profile = profile.name
			reportHiddenWatermarkProgress(progress, value)
		}
		var groups map[int][][]byte
		var err error
		if progress == nil {
			groups, err = p.remote.ExtractCandidateGroupsWithReference(remoteCtx, data, referenceData, payloadSizes, options)
		} else {
			groups, err = p.remote.ExtractCandidateGroupsWithReferenceProgress(remoteCtx, data, referenceData, payloadSizes, options, profileProgress)
		}
		if err != nil {
			lastErr = err
			continue
		}
		succeeded = true
		for _, spec := range specs {
			candidates := groups[spec.encodedSize]
			for _, extracted := range candidates {
				result := p.decodeExtractedWatermarkPayload(ctx, extracted, spec.payloadSize)
				if result.Found {
					return result, nil
				}
			}
		}
	}
	if !succeeded && lastErr != nil {
		return HiddenWatermarkData{}, lastErr
	}
	return HiddenWatermarkData{Found: false}, nil
}

func (p *ImageProcessor) verifyExpectedTraceToken(ctx context.Context, data, referenceData []byte, traceToken, engine string, input ProcessImageInput) (HiddenWatermarkData, error) {
	if engine != "remote" {
		result, err := p.ExtractWithReference(ctx, data, referenceData)
		if err != nil {
			return result, err
		}
		if result.TraceToken != traceToken {
			return HiddenWatermarkData{Found: false}, errors.New("watermark payload mismatch")
		}
		return result, nil
	}
	if p.remote == nil || !p.remote.Configured() {
		return HiddenWatermarkData{}, errors.New("hidden watermark remote service is not configured")
	}
	remoteCtx, cancel := p.remoteWatermarkRequestContext(ctx)
	defer cancel()
	options := p.remoteWatermarkOptionsForExtraction(input)
	options.RecoverDimensions = p.remoteWatermarkRecoverDimensions(ctx, data, referenceData)
	groups, err := p.remote.ExtractCandidateGroupsWithReference(
		remoteCtx,
		data,
		referenceData,
		hiddenWatermarkEncodedPayloadSizes(p.hiddenWatermarkExtractSpecs()),
		options,
	)
	if err != nil {
		return HiddenWatermarkData{}, err
	}
	for _, spec := range p.hiddenWatermarkExtractSpecs() {
		for _, candidate := range groups[spec.encodedSize] {
			result := p.decodeExtractedWatermarkPayload(ctx, candidate, spec.payloadSize)
			if result.Found && result.Valid && result.TraceToken == traceToken {
				return result, nil
			}
		}
	}
	return HiddenWatermarkData{Found: false}, errors.New("watermark payload mismatch")
}

func reportHiddenWatermarkProgress(progress HiddenWatermarkProgressFunc, value HiddenWatermarkProgress) {
	if progress == nil {
		return
	}
	value.Percent = min(max(value.Percent, 0), 100)
	progress(value)
}

func hiddenWatermarkEmbeddingPayloadSize(input ProcessImageInput) int {
	if strings.TrimSpace(input.WatermarkPayloadToken) != "" {
		if decoded, err := hex.DecodeString(strings.TrimSpace(input.WatermarkPayloadToken)); err == nil && len(decoded) > 0 {
			return len(decoded)
		}
	}
	return domain.ImageWatermarkShortCodeBytes
}

func (p *ImageProcessor) hiddenWatermarkExtractSpecs() []hiddenWatermarkExtractSpec {
	return []hiddenWatermarkExtractSpec{
		{
			payloadSize: domain.ImageWatermarkShortCodeBytes,
			encodedSize: domain.ImageWatermarkShortCodeBytes,
		},
		{
			payloadSize: 3,
			encodedSize: 3,
		},
		{
			payloadSize: domain.ImageWatermarkMinShortCodeBytes,
			encodedSize: domain.ImageWatermarkMinShortCodeBytes,
		},
	}
}

func hiddenWatermarkEncodedPayloadSizes(specs []hiddenWatermarkExtractSpec) []int {
	values := make([]int, 0, len(specs))
	seen := map[int]bool{}
	for _, spec := range specs {
		if spec.encodedSize <= 0 || seen[spec.encodedSize] {
			continue
		}
		seen[spec.encodedSize] = true
		values = append(values, spec.encodedSize)
	}
	return values
}

func (p *ImageProcessor) decodeExtractedWatermarkPayload(ctx context.Context, extracted []byte, payloadSize int) HiddenWatermarkData {
	if payloadSize >= domain.ImageWatermarkMinShortCodeBytes && payloadSize <= domain.ImageWatermarkShortCodeBytes && len(extracted) == payloadSize && p.payloadResolver != nil {
		result := p.payloadResolver(ctx, extracted)
		if result.Found && result.Valid {
			result.PayloadBytes = payloadSize
			result.PayloadBits = payloadSize * 8
			if result.PayloadFormat == "" {
				result.PayloadFormat = "short_code_v1"
			}
		}
		return result
	}
	return HiddenWatermarkData{Found: false}
}
