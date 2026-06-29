//go:build !cgo

package services

import "image"

func decodeWithLibvips([]byte) (image.Image, imageColorMetadata, error) {
	return nil, imageColorMetadata{}, errColorPipelineUnavailable
}

func encodeWebPWithLibvips(image.Image, imageColorMetadata, colorManagedWebPOptions) ([]byte, error) {
	return nil, errColorPipelineUnavailable
}
