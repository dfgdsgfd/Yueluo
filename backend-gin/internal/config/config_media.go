package config

import (
	"strings"
)

func loadWebPConfig() WebPConfig {
	return WebPConfig{
		EnableConversion: boolEnv("WEBP_ENABLE_CONVERSION", true),
		Quality:          intEnv("WEBP_QUALITY", 85),
		AvatarQuality:    intEnv("AVATAR_WEBP_QUALITY", 75),
		Method:           intEnv("WEBP_METHOD", 4),
		ConvertJPEG:      boolEnv("WEBP_CONVERT_JPEG", true),
		ConvertPNG:       boolEnv("WEBP_CONVERT_PNG", true),
		KeepOriginal:     boolEnv("WEBP_KEEP_ORIGINAL", false),
		MaxWidth:         optionalIntEnv("WEBP_MAX_WIDTH"),
		MaxHeight:        optionalIntEnv("WEBP_MAX_HEIGHT"),
		Lossless:         boolEnv("WEBP_LOSSLESS", false),
		AlphaQuality:     intEnv("WEBP_ALPHA_QUALITY", 100),
		HiddenWatermark: HiddenWatermarkConfig{
			Secret: getEnv("HIDDEN_WATERMARK_SECRET", ""),
			Remote: HiddenWatermarkRemoteConfig{
				URL:           strings.TrimRight(getEnv("HIDDEN_WATERMARK_REMOTE_URL", ""), "/"),
				APIKey:        getEnv("HIDDEN_WATERMARK_REMOTE_API_KEY", ""),
				SkipTLSVerify: boolEnv("HIDDEN_WATERMARK_REMOTE_SKIP_TLS_VERIFY", false),
			},
		},
		Watermark: WatermarkConfig{
			Enabled:      boolEnv("WATERMARK_ENABLED", false),
			Type:         getEnv("WATERMARK_TYPE", "text"),
			Text:         getEnv("WATERMARK_TEXT", ""),
			FontSize:     intEnv("WATERMARK_FONT_SIZE", 24),
			FontPath:     getEnv("WATERMARK_FONT_PATH", ""),
			ImagePath:    getEnv("WATERMARK_IMAGE_PATH", ""),
			Opacity:      intEnv("WATERMARK_OPACITY", 50),
			Position:     getEnv("WATERMARK_POSITION", "9"),
			PositionMode: getEnv("WATERMARK_POSITION_MODE", "grid"),
			PreciseX:     intEnv("WATERMARK_PRECISE_X", 0),
			PreciseY:     intEnv("WATERMARK_PRECISE_Y", 0),
			ImageRatio:   intEnv("WATERMARK_IMAGE_RATIO", 4),
			TileMode:     boolEnv("WATERMARK_TILE_MODE", false),
			Color:        getEnv("WATERMARK_COLOR", "#ffffff"),
		},
		UsernameWatermark: WatermarkConfig{
			Enabled:      boolEnv("USERNAME_WATERMARK_ENABLED", false),
			Type:         "text",
			Text:         getEnv("USERNAME_WATERMARK_TEXT", "@username"),
			FontSize:     intEnv("USERNAME_WATERMARK_FONT_SIZE", 20),
			FontPath:     getEnv("USERNAME_WATERMARK_FONT_PATH", ""),
			Opacity:      intEnv("USERNAME_WATERMARK_OPACITY", 70),
			Position:     getEnv("USERNAME_WATERMARK_POSITION", "7"),
			PositionMode: getEnv("USERNAME_WATERMARK_POSITION_MODE", "grid"),
			PreciseX:     intEnv("USERNAME_WATERMARK_PRECISE_X", 20),
			PreciseY:     intEnv("USERNAME_WATERMARK_PRECISE_Y", 20),
			Color:        getEnv("USERNAME_WATERMARK_COLOR", "#ffffff"),
		},
	}
}

func loadVideoTranscodingConfig() VideoTranscodingConfig {
	return VideoTranscodingConfig{
		Enabled:            boolEnv("VIDEO_TRANSCODING_ENABLED", false),
		FFmpegPath:         getEnv("FFMPEG_PATH", "/app/bin/ffmpeg"),
		FFprobePath:        getEnv("FFPROBE_PATH", "/app/bin/ffprobe"),
		MaxThreads:         intEnv("VIDEO_TRANSCODING_MAX_THREADS", 4),
		MaxConcurrentTasks: intEnv("VIDEO_TRANSCODING_MAX_CONCURRENT", 2),
		OutputFormat:       getEnv("VIDEO_DASH_OUTPUT_FORMAT", "{date}/{userId}/{timestamp}"),
		DASH: DASHConfig{
			SegmentDuration:    intEnv("DASH_SEGMENT_DURATION", 4),
			MinBitrate:         intEnv("DASH_MIN_BITRATE", 500),
			MaxBitrate:         intEnv("DASH_MAX_BITRATE", 5000),
			OriginalMaxBitrate: intEnv("ORIGINAL_VIDEO_MAX_BITRATE", 8000),
			Resolutions:        parseDASHResolutions(getEnv("DASH_RESOLUTIONS", "1920x1080:5000,1280x720:2500,854x480:1000,640x360:750")),
		},
		DeleteOriginal: boolEnv("DELETE_ORIGINAL_VIDEO", false),
		FFmpeg: FFmpegConfig{
			Preset:            getEnv("FFMPEG_PRESET", "medium"),
			Profile:           getEnv("FFMPEG_PROFILE", "main"),
			CRF:               optionalRangedIntEnv("FFMPEG_CRF", 10, 51),
			GOPSize:           optionalIntEnv("FFMPEG_GOP_SIZE"),
			BFrames:           optionalIntEnv("FFMPEG_B_FRAMES"),
			RefFrames:         optionalIntEnv("FFMPEG_REF_FRAMES"),
			Complexity:        optionalIntEnv("FFMPEG_COMPLEXITY"),
			AudioBitrate:      intEnv("FFMPEG_AUDIO_BITRATE", 128),
			AudioSampleRate:   intEnv("FFMPEG_AUDIO_SAMPLE_RATE", 48000),
			PixelFormat:       getEnv("FFMPEG_PIXEL_FORMAT", "yuv420p"),
			HardwareAccel:     boolEnv("FFMPEG_HARDWARE_ACCEL", false),
			HardwareAccelType: getEnv("FFMPEG_HARDWARE_ACCEL_TYPE", ""),
		},
	}
}

func loadR2Config() R2Config {
	return R2Config{
		AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		BucketName:      getEnv("R2_BUCKET_NAME", ""),
		Endpoint:        strings.TrimRight(getEnv("R2_ENDPOINT", ""), "/"),
		PublicURL:       strings.TrimRight(getEnv("R2_PUBLIC_URL", ""), "/"),
		Region:          getEnv("R2_REGION", "auto"),
	}
}
