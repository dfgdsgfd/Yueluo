package services

import "testing"

func TestFilterSafeEXIFDropsSensitiveMetadata(t *testing.T) {
	filtered := filterSafeEXIF(map[string]string{
		"exif-ifd0-Model":            "TraceCam",
		"exif-ifd0-Copyright":        "Copyright",
		"exif-ifd2-DateTimeOriginal": "2026:06:18 12:00:00",
		"exif-ifd2-BodySerialNumber": "camera-secret",
		"exif-ifd2-LensSerialNumber": "lens-secret",
		"exif-ifd3-GPSLatitude":      "31.2304",
		"exif-ifd3-GPSLongitude":     "121.4737",
		"exif-ifd0-ImageUniqueID":    "unique-secret",
	})
	for _, key := range []string{"exif-ifd0-Model", "exif-ifd0-Copyright", "exif-ifd2-DateTimeOriginal"} {
		if filtered[key] == "" {
			t.Fatalf("safe EXIF field %q was removed", key)
		}
	}
	for _, key := range []string{"exif-ifd2-BodySerialNumber", "exif-ifd2-LensSerialNumber", "exif-ifd3-GPSLatitude", "exif-ifd3-GPSLongitude", "exif-ifd0-ImageUniqueID"} {
		if _, ok := filtered[key]; ok {
			t.Fatalf("sensitive EXIF field %q was retained", key)
		}
	}
}
