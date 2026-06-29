package handlers

import (
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) notificationSuppressionConfig() repositories.NotificationSuppressionConfig {
	settings := h.Settings
	if settings == nil {
		settings = services.NewSettingsService(nil, nil)
	}
	return repositories.NotificationSuppressionConfig{
		Enabled:       settings.Bool("notification_interaction_suppression_enabled"),
		WindowSeconds: settings.Int("notification_interaction_suppression_window_seconds", services.DefaultNotificationInteractionSuppressionWindowSeconds),
		Threshold:     settings.Int("notification_interaction_suppression_threshold", services.DefaultNotificationInteractionSuppressionThreshold),
	}
}
