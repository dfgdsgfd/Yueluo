package services

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
)

func encodePNGImage(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *ImageProcessor) useRemoteHiddenWatermark() bool {
	if p != nil && p.requireRemote {
		return true
	}
	engine := p.hiddenWatermarkEngine()
	return engine == "remote" || engine == "auto"
}

func (p *ImageProcessor) hiddenWatermarkAttemptCount() int {
	if p.useRemoteHiddenWatermark() && p.remote != nil && p.remote.Configured() {
		return len(p.remoteWatermarkEmbedAttemptsForInput(ProcessImageInput{}))
	}
	return len(p.hiddenWatermarkProfiles())
}

func (p *ImageProcessor) hiddenWatermarkAttemptCountForInput(input ProcessImageInput) int {
	if p.requireRemote || (p.useRemoteHiddenWatermark() && p.remote != nil && p.remote.Configured()) {
		return len(p.remoteWatermarkEmbedAttemptsForInput(input))
	}
	return len(p.hiddenWatermarkProfilesForInput(input))
}

func (p *ImageProcessor) hiddenWatermarkEngine() string {
	switch strings.ToLower(strings.TrimSpace(p.stringSetting("hidden_watermark_engine"))) {
	case "local", "remote":
		return strings.ToLower(strings.TrimSpace(p.stringSetting("hidden_watermark_engine")))
	default:
		return "auto"
	}
}

func (p *ImageProcessor) hiddenWatermarkProfiles() []hiddenWatermarkProfile {
	configured := p.configuredHiddenWatermarkProfile()
	profiles := []hiddenWatermarkProfile{configured}
	for _, profile := range hiddenWatermarkProfiles {
		if !containsHiddenWatermarkProfile(profiles, profile) {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func (p *ImageProcessor) hiddenWatermarkProfilesForInput(input ProcessImageInput) []hiddenWatermarkProfile {
	profiles := p.hiddenWatermarkProfiles()
	if !input.PreferRobust {
		return profiles
	}
	robust, _ := hiddenWatermarkPresetProfile("robust")
	reordered := []hiddenWatermarkProfile{robust}
	for _, profile := range profiles {
		if profile != robust {
			reordered = append(reordered, profile)
		}
	}
	return reordered
}

func (p *ImageProcessor) hiddenWatermarkExtractProfiles() []hiddenWatermarkProfile {
	return p.hiddenWatermarkProfiles()
}

func (p *ImageProcessor) configuredHiddenWatermarkProfile() hiddenWatermarkProfile {
	fallback := hiddenWatermarkProfiles[0]
	if preset, ok := hiddenWatermarkPresetProfile(p.stringSetting("hidden_watermark_profile")); ok {
		return preset
	}
	coefficientMode := p.stringSetting("hidden_watermark_coefficient_mode")
	if coefficientMode != "d1" && coefficientMode != "d1d2" {
		coefficientMode = fallback.coefficientMode
	}
	eccMode := p.stringSetting("hidden_watermark_ecc_mode")
	if eccMode != "none" && eccMode != "golay" {
		eccMode = fallback.eccMode
	}
	profile := hiddenWatermarkProfile{
		blockWidth:      normalizeHiddenWatermarkBlock(p.intSetting("hidden_watermark_block_width", fallback.blockWidth, 4, 64)),
		blockHeight:     normalizeHiddenWatermarkBlock(p.intSetting("hidden_watermark_block_height", fallback.blockHeight, 4, 64)),
		coefficientMode: coefficientMode,
		d1:              p.intSetting("hidden_watermark_d1", fallback.d1, 1, 64),
		d2:              p.intSetting("hidden_watermark_d2", fallback.d2, 0, 64),
		eccMode:         eccMode,
		golaySeed:       int64(p.intSetting("hidden_watermark_golay_seed", int(mark.DefaultShuffleSeed), 0, 2_147_483_647)),
	}
	if profile.d2 > profile.d1 {
		profile.d2 = profile.d1
	}
	return profile
}

func (p *ImageProcessor) remoteWatermarkOptions(input ProcessImageInput) HiddenWatermarkRemoteOptions {
	attempts := p.remoteWatermarkEmbedAttemptsForInput(input)
	attempt := attempts[min(max(input.WatermarkAttempt, 0), len(attempts)-1)]
	options := p.remoteWatermarkOptionsForProfile(attempt.profile)
	options.Engine = attempt.engine
	return options
}

func (p *ImageProcessor) remoteWatermarkOptionsForImage(img image.Image, input ProcessImageInput) HiddenWatermarkRemoteOptions {
	options := p.remoteWatermarkOptions(input)
	if img != nil {
		bounds := img.Bounds()
		options.WatermarkWidth = bounds.Dx()
		options.WatermarkHeight = bounds.Dy()
		options.RecoverDimensions = appendRemoteWatermarkRecoverDimensions(options.RecoverDimensions, [2]int{bounds.Dx(), bounds.Dy()})
	}
	return options
}

func (p *ImageProcessor) remoteWatermarkOptionsForExtraction(input ProcessImageInput) HiddenWatermarkRemoteOptions {
	options := p.remoteWatermarkOptions(input)
	return options
}

func (p *ImageProcessor) remoteWatermarkOptionsForProfile(profile hiddenWatermarkRemoteProfile) HiddenWatermarkRemoteOptions {
	return HiddenWatermarkRemoteOptions{
		PasswordWM:              profile.passwordWM,
		PasswordImg:             profile.passwordImg,
		D1:                      profile.d1,
		D2:                      profile.d2,
		Profile:                 profile.name,
		OperationTimeoutSeconds: p.remoteWatermarkOperationTimeoutSeconds(),
		Engine:                  p.remoteWatermarkEngine(),
		DWTDctSVDRepeat:         p.intSetting("hidden_watermark_remote_dwt_repeat", 9, 1, 21),
		DWTDctSVDScale:          p.intSetting("hidden_watermark_remote_dwt_scale", 64, 1, 128),
	}
}

type hiddenWatermarkRemoteAttempt struct {
	profile hiddenWatermarkRemoteProfile
	engine  string
}

func (p *ImageProcessor) remoteWatermarkEngine() string {
	switch strings.ToLower(strings.TrimSpace(p.stringSetting("hidden_watermark_remote_engine"))) {
	case "auto", "blind_watermark", "dwt_dct_svd":
		return strings.ToLower(strings.TrimSpace(p.stringSetting("hidden_watermark_remote_engine")))
	default:
		return "blind_watermark"
	}
}

func (p *ImageProcessor) remoteWatermarkRecoverDimensions(ctx context.Context, data, referenceData []byte) [][2]int {
	dimensions := remoteWatermarkRecoverDimensionsForData(data, referenceData)
	if p != nil && p.dimensionResolver != nil {
		dimensions = appendRemoteWatermarkRecoverDimensions(dimensions, p.dimensionResolver(ctx)...)
	}
	return dimensions
}

func remoteWatermarkRecoverDimensionsForData(data, referenceData []byte) [][2]int {
	dimensions := make([][2]int, 0, 6)
	if len(referenceData) > 0 {
		if _, cfg, err := inspectImage(referenceData); err == nil {
			dimensions = appendRemoteWatermarkRecoverDimensions(dimensions, [2]int{cfg.Width, cfg.Height})
		}
	}
	if _, cfg, err := inspectImage(data); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		dimensions = appendRemoteWatermarkRecoverDimensions(dimensions, [2]int{cfg.Width, cfg.Height})
		longest := max(cfg.Width, cfg.Height)
		for _, targetLongest := range []int{720, 960, 1280, 2048} {
			if longest <= 0 {
				continue
			}
			scale := float64(targetLongest) / float64(longest)
			dimensions = appendRemoteWatermarkRecoverDimensions(dimensions, [2]int{
				max(1, int(math.Round(float64(cfg.Width)*scale))),
				max(1, int(math.Round(float64(cfg.Height)*scale))),
			})
		}
	}
	return dimensions
}

func appendRemoteWatermarkRecoverDimensions(dimensions [][2]int, candidates ...[2]int) [][2]int {
	for _, candidate := range candidates {
		if candidate[0] <= 0 || candidate[1] <= 0 {
			continue
		}
		seen := slices.Contains(dimensions, candidate)
		if !seen {
			dimensions = append(dimensions, candidate)
		}
	}
	return dimensions
}

func (p *ImageProcessor) remoteWatermarkRequestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, time.Duration(p.remoteWatermarkTimeoutSeconds())*time.Second)
}

func (p *ImageProcessor) remoteWatermarkTimeoutSeconds() int {
	return p.intSetting("hidden_watermark_remote_timeout_seconds", 50, 10, 300)
}

func (p *ImageProcessor) remoteWatermarkOperationTimeoutSeconds() int {
	return p.intSetting("hidden_watermark_remote_operation_timeout_seconds", 45, 10, 300)
}

func (p *ImageProcessor) configuredHiddenWatermarkRemoteProfile() hiddenWatermarkRemoteProfile {
	profile := hiddenWatermarkRemoteProfile{
		name:        "custom",
		passwordWM:  p.intSetting("hidden_watermark_remote_password_wm", officialHiddenWatermarkRemoteProfile.passwordWM, 0, 2_147_483_647),
		passwordImg: p.intSetting("hidden_watermark_remote_password_img", officialHiddenWatermarkRemoteProfile.passwordImg, 0, 2_147_483_647),
		d1:          p.intSetting("hidden_watermark_remote_custom_d1", 18, 1, 64),
		d2:          p.intSetting("hidden_watermark_remote_custom_d2", 8, 0, 64),
	}
	if profile.d2 > profile.d1 {
		profile.d2 = profile.d1
	}
	return profile
}

func (p *ImageProcessor) remoteWatermarkEmbedProfiles() []hiddenWatermarkRemoteProfile {
	return p.remoteWatermarkEmbedProfilesForPreference(false)
}

func (p *ImageProcessor) remoteWatermarkEmbedProfilesForInput(input ProcessImageInput) []hiddenWatermarkRemoteProfile {
	if input.ForceRemoteAdaptive {
		return p.remoteWatermarkAdaptiveProfilesForPreference(input.PreferRobust)
	}
	return p.remoteWatermarkEmbedProfilesForPreference(input.PreferRobust)
}

func (p *ImageProcessor) remoteWatermarkEmbedAttemptsForInput(input ProcessImageInput) []hiddenWatermarkRemoteAttempt {
	profiles := p.remoteWatermarkEmbedProfilesForInput(input)
	engine := p.remoteWatermarkEngine()
	if engine != "auto" {
		return remoteWatermarkAttemptsForEngine(profiles, engine)
	}
	attempts := remoteWatermarkAttemptsForEngine(profiles, "blind_watermark")
	dwtProfile := p.configuredHiddenWatermarkRemoteProfile()
	if len(profiles) > 0 {
		dwtProfile = profiles[len(profiles)-1]
	}
	return append(attempts, hiddenWatermarkRemoteAttempt{
		profile: dwtProfile,
		engine:  "dwt_dct_svd",
	})
}

func remoteWatermarkAttemptsForEngine(profiles []hiddenWatermarkRemoteProfile, engine string) []hiddenWatermarkRemoteAttempt {
	attempts := make([]hiddenWatermarkRemoteAttempt, 0, len(profiles))
	for _, profile := range profiles {
		attempts = append(attempts, hiddenWatermarkRemoteAttempt{profile: profile, engine: engine})
	}
	return attempts
}

func (p *ImageProcessor) remoteWatermarkEmbedProfilesForPreference(preferRobust bool) []hiddenWatermarkRemoteProfile {
	selected := strings.ToLower(strings.TrimSpace(p.stringSetting("hidden_watermark_remote_profile")))
	passwordWM := p.intSetting("hidden_watermark_remote_password_wm", officialHiddenWatermarkRemoteProfile.passwordWM, 0, 2_147_483_647)
	passwordImg := p.intSetting("hidden_watermark_remote_password_img", officialHiddenWatermarkRemoteProfile.passwordImg, 0, 2_147_483_647)
	if selected == "" || selected == "adaptive" {
		profiles := append([]hiddenWatermarkRemoteProfile(nil), hiddenWatermarkRemoteStrengthProfiles[:]...)
		if preferRobust {
			profiles = skipRemoteFidelityProfile(profiles)
		}
		for i := range profiles {
			profiles[i].passwordWM = passwordWM
			profiles[i].passwordImg = passwordImg
		}
		return profiles
	}
	if selected == "custom" {
		return []hiddenWatermarkRemoteProfile{p.configuredHiddenWatermarkRemoteProfile()}
	}
	for _, profile := range hiddenWatermarkRemoteStrengthProfiles {
		if profile.name == selected {
			if preferRobust && profile.name == "fidelity" {
				profile = hiddenWatermarkRemoteStrengthProfiles[1]
			}
			profile.passwordWM = passwordWM
			profile.passwordImg = passwordImg
			return []hiddenWatermarkRemoteProfile{profile}
		}
	}
	profiles := append([]hiddenWatermarkRemoteProfile(nil), hiddenWatermarkRemoteStrengthProfiles[:]...)
	if preferRobust {
		profiles = skipRemoteFidelityProfile(profiles)
	}
	for i := range profiles {
		profiles[i].passwordWM = passwordWM
		profiles[i].passwordImg = passwordImg
	}
	return profiles
}

func (p *ImageProcessor) remoteWatermarkAdaptiveProfilesForPreference(preferRobust bool) []hiddenWatermarkRemoteProfile {
	profiles := append([]hiddenWatermarkRemoteProfile(nil), hiddenWatermarkRemoteStrengthProfiles[:]...)
	if preferRobust {
		profiles = skipRemoteFidelityProfile(profiles)
	}
	passwordWM := p.intSetting("hidden_watermark_remote_password_wm", officialHiddenWatermarkRemoteProfile.passwordWM, 0, 2_147_483_647)
	passwordImg := p.intSetting("hidden_watermark_remote_password_img", officialHiddenWatermarkRemoteProfile.passwordImg, 0, 2_147_483_647)
	for i := range profiles {
		profiles[i].passwordWM = passwordWM
		profiles[i].passwordImg = passwordImg
	}
	return profiles
}

func skipRemoteFidelityProfile(profiles []hiddenWatermarkRemoteProfile) []hiddenWatermarkRemoteProfile {
	filtered := make([]hiddenWatermarkRemoteProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profile.name != "fidelity" {
			filtered = append(filtered, profile)
		}
	}
	if len(filtered) == 0 {
		return profiles
	}
	return filtered
}

func (p *ImageProcessor) remoteWatermarkExtractProfiles() []hiddenWatermarkRemoteProfile {
	profiles := []hiddenWatermarkRemoteProfile{p.configuredHiddenWatermarkRemoteProfile()}
	for _, profile := range hiddenWatermarkRemoteStrengthProfiles {
		profile.passwordWM = p.intSetting("hidden_watermark_remote_password_wm", profile.passwordWM, 0, 2_147_483_647)
		profile.passwordImg = p.intSetting("hidden_watermark_remote_password_img", profile.passwordImg, 0, 2_147_483_647)
		if !containsHiddenWatermarkRemoteProfile(profiles, profile) {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func containsHiddenWatermarkRemoteProfile(profiles []hiddenWatermarkRemoteProfile, target hiddenWatermarkRemoteProfile) bool {
	for _, profile := range profiles {
		if profile.passwordWM == target.passwordWM && profile.passwordImg == target.passwordImg &&
			profile.d1 == target.d1 && profile.d2 == target.d2 {
			return true
		}
	}
	return false
}

func hiddenWatermarkPresetProfile(name string) (hiddenWatermarkProfile, bool) {
	switch strings.TrimSpace(name) {
	case "current":
		return hiddenWatermarkProfiles[0], true
	case "author_recommended":
		return hiddenWatermarkProfile{blockWidth: 8, blockHeight: 8, coefficientMode: "d1d2", d1: 21, d2: 9, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed}, true
	case "fidelity":
		return hiddenWatermarkProfile{blockWidth: 12, blockHeight: 12, coefficientMode: "d1d2", d1: 8, d2: 3, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed}, true
	case "robust":
		return hiddenWatermarkProfile{blockWidth: 6, blockHeight: 4, coefficientMode: "d1d2", d1: 36, d2: 20, eccMode: "golay", golaySeed: mark.DefaultShuffleSeed}, true
	default:
		return hiddenWatermarkProfile{}, false
	}
}

func watermarkOptionsForHiddenWatermarkProfile(profile hiddenWatermarkProfile) []watermark.Option {
	options := []watermark.Option{watermark.WithBlockShape(profile.blockWidth, profile.blockHeight)}
	if profile.coefficientMode == "d1" {
		return append(options, watermark.WithD1(profile.d1))
	}
	return append(options, watermark.WithD1D2(profile.d1, profile.d2))
}

func markOptionsForHiddenWatermarkProfile(profile hiddenWatermarkProfile) []mark.Option {
	if profile.eccMode == "none" {
		return []mark.Option{mark.WithoutECC()}
	}
	return []mark.Option{mark.WithGolay(profile.golaySeed)}
}

func normalizeHiddenWatermarkBlock(value int) int {
	if value < 4 {
		return 4
	}
	if value%2 != 0 {
		value++
	}
	return value
}

func containsHiddenWatermarkProfile(profiles []hiddenWatermarkProfile, target hiddenWatermarkProfile) bool {
	return slices.Contains(profiles, target)
}

func ensureHiddenWatermarkCapacity(img image.Image, profile hiddenWatermarkProfile, requiredBits int) (image.Image, error) {
	if img == nil || requiredBits <= 0 || hiddenWatermarkCapacity(img.Bounds(), profile) >= requiredBits {
		return img, nil
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, ErrImageInvalid
	}
	targetArea := float64(requiredBits * profile.blockWidth * profile.blockHeight)
	scale := math.Max(1.1, math.Sqrt(targetArea/float64(width*height))*1.1)
	newWidth := int(math.Ceil(float64(width) * scale))
	newHeight := int(math.Ceil(float64(height) * scale))
	for hiddenWatermarkCapacity(image.Rect(0, 0, newWidth, newHeight), profile) < requiredBits {
		newWidth = int(math.Ceil(float64(newWidth) * 1.05))
		newHeight = int(math.Ceil(float64(newHeight) * 1.05))
	}
	if newWidth > maxDecodedImageDimension || newHeight > maxDecodedImageDimension || int64(newWidth)*int64(newHeight) > maxDecodedImagePixels {
		return nil, ErrImageTooLarge
	}
	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos), nil
}
