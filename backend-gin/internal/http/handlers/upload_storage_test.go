package handlers

import (
	"testing"

	"yuem-go/backend-gin/internal/services"
)

func TestUploadAssetPurposeIsTemporary(t *testing.T) {
	temporary := []services.ImagePurpose{
		services.ImagePurposeContent,
		services.ImagePurposeCover,
		services.ImagePurposeAIAnalysis,
	}
	for _, purpose := range temporary {
		if !uploadAssetPurposeIsTemporary(purpose) {
			t.Fatalf("expected %s to be tracked as a temporary upload asset", purpose)
		}
	}

	permanent := []services.ImagePurpose{
		services.ImagePurposeAvatar,
		services.ImagePurposeBackground,
		services.ImagePurposeFeedback,
	}
	for _, purpose := range permanent {
		if uploadAssetPurposeIsTemporary(purpose) {
			t.Fatalf("expected %s to skip temporary upload asset tracking", purpose)
		}
	}
}
