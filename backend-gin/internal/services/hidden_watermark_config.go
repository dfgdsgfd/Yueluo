package services

import "yuem-go/backend-gin/internal/config"

func HiddenWatermarkRemoteClientConfigFromConfig(cfg config.Config) HiddenWatermarkRemoteClientConfig {
	remote := cfg.WebP.HiddenWatermark.Remote
	return HiddenWatermarkRemoteClientConfig{
		URL:           remote.URL,
		APIKey:        remote.APIKey,
		SkipTLSVerify: remote.SkipTLSVerify,
	}
}
