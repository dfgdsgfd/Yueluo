package services

import (
	"errors"
	"image"
)

var errColorPipelineUnavailable = errors.New("libvips color pipeline is unavailable")

type imageColorMetadata struct {
	ICCProfile []byte
	SafeEXIF   map[string]string
}

type colorManagedWebPOptions struct {
	AlphaQuality int
	Effort       int
	Lossless     bool
	Quality      int
}

var safeEXIFFields = map[string]struct{}{
	"exif-ifd0-Artist":                  {},
	"exif-ifd0-Copyright":               {},
	"exif-ifd0-DateTime":                {},
	"exif-ifd0-ImageDescription":        {},
	"exif-ifd0-Make":                    {},
	"exif-ifd0-Model":                   {},
	"exif-ifd0-Software":                {},
	"exif-ifd2-ApertureValue":           {},
	"exif-ifd2-ColorSpace":              {},
	"exif-ifd2-Contrast":                {},
	"exif-ifd2-DateTimeDigitized":       {},
	"exif-ifd2-DateTimeOriginal":        {},
	"exif-ifd2-DigitalZoomRatio":        {},
	"exif-ifd2-ExposureBiasValue":       {},
	"exif-ifd2-ExposureProgram":         {},
	"exif-ifd2-ExposureTime":            {},
	"exif-ifd2-FNumber":                 {},
	"exif-ifd2-Flash":                   {},
	"exif-ifd2-FocalLength":             {},
	"exif-ifd2-FocalLengthIn35mmFilm":   {},
	"exif-ifd2-ISOSpeedRatings":         {},
	"exif-ifd2-LensMake":                {},
	"exif-ifd2-LensModel":               {},
	"exif-ifd2-LightSource":             {},
	"exif-ifd2-MeteringMode":            {},
	"exif-ifd2-PhotographicSensitivity": {},
	"exif-ifd2-Saturation":              {},
	"exif-ifd2-SceneCaptureType":        {},
	"exif-ifd2-Sharpness":               {},
	"exif-ifd2-ShutterSpeedValue":       {},
	"exif-ifd2-WhiteBalance":            {},
}

func filterSafeEXIF(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	const maxFieldBytes = 1024
	const maxTotalBytes = 16 * 1024
	result := make(map[string]string)
	total := 0
	for name, value := range source {
		if _, ok := safeEXIFFields[name]; !ok || value == "" || len(value) > maxFieldBytes {
			continue
		}
		if total+len(name)+len(value) > maxTotalBytes {
			continue
		}
		result[name] = value
		total += len(name) + len(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func decodeWithColorManagement(data []byte) (image.Image, imageColorMetadata, error) {
	return decodeWithLibvips(data)
}

func encodeColorManagedWebP(img image.Image, metadata imageColorMetadata, options colorManagedWebPOptions) ([]byte, error) {
	return encodeWebPWithLibvips(img, metadata, options)
}
