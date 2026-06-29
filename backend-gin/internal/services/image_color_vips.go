//go:build cgo

package services

import (
	"image"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

var (
	libvipsStartOnce sync.Once
	libvipsStartErr  error
)

func ensureLibvipsStarted() error {
	libvipsStartOnce.Do(func() {
		vips.LoggingSettings(nil, vips.LogLevelWarning)
		libvipsStartErr = vips.Startup(&vips.Config{
			ConcurrencyLevel: 1,
			MaxCacheFiles:    -1,
			MaxCacheMem:      -1,
			MaxCacheSize:     -1,
		})
	})
	return libvipsStartErr
}

func decodeWithLibvips(data []byte) (image.Image, imageColorMetadata, error) {
	if err := ensureLibvipsStarted(); err != nil {
		return nil, imageColorMetadata{}, err
	}
	params := vips.NewImportParams()
	params.AutoRotate.Set(true)
	ref, err := vips.LoadImageFromBuffer(data, params)
	if err != nil {
		return nil, imageColorMetadata{}, err
	}
	defer ref.Close()

	metadata := imageColorMetadata{SafeEXIF: filterSafeEXIF(ref.GetExif())}
	if err := ref.TransformICCProfileWithFallback(vips.SRGBV2MicroICCProfilePath, vips.SRGBV2MicroICCProfilePath); err != nil {
		return nil, imageColorMetadata{}, err
	}
	metadata.ICCProfile = append([]byte(nil), ref.GetICCProfile()...)
	img, err := ref.ToGoImage()
	if err != nil {
		return nil, imageColorMetadata{}, err
	}
	return img, metadata, nil
}

func encodeWebPWithLibvips(img image.Image, metadata imageColorMetadata, options colorManagedWebPOptions) ([]byte, error) {
	if err := ensureLibvipsStarted(); err != nil {
		return nil, err
	}
	if options.AlphaQuality < 100 {
		img = quantizeImageAlpha(img, options.AlphaQuality)
	}
	ref, err := vips.NewImageFromGoImage(img)
	if err != nil {
		return nil, err
	}
	defer ref.Close()

	if err := ref.TransformICCProfileWithFallback(vips.SRGBV2MicroICCProfilePath, vips.SRGBV2MicroICCProfilePath); err != nil {
		return nil, err
	}
	if err := ref.OptimizeICCProfile(); err != nil {
		return nil, err
	}
	for name, value := range metadata.SafeEXIF {
		ref.SetString(name, value)
	}
	if err := ref.SetOrientation(1); err != nil {
		return nil, err
	}

	params := vips.NewWebpExportParams()
	params.Lossless = options.Lossless
	params.NearLossless = false
	params.Quality = options.Quality
	params.ReductionEffort = options.Effort
	params.StripMetadata = false
	data, _, err := ref.ExportWebp(params)
	return data, err
}
