package services

import (
	"context"
	"testing"
	"time"
)

func TestMaintenanceBypassSignature(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	value := MaintenanceBypassValue("entry-code", "secret", now)

	if !ValidMaintenanceBypass(value, "entry-code", "secret") {
		t.Fatalf("ValidMaintenanceBypass() = false, want true")
	}
	if ValidMaintenanceBypass(value, "other-code", "secret") {
		t.Fatalf("ValidMaintenanceBypass accepted wrong entry code")
	}
	if ValidMaintenanceBypass(value, "entry-code", "other-secret") {
		t.Fatalf("ValidMaintenanceBypass accepted wrong secret")
	}
	if ValidMaintenanceBypass("entry-code:bad", "entry-code", "secret") {
		t.Fatalf("ValidMaintenanceBypass accepted malformed value")
	}
}

func TestMaintenanceStateIncludesBorderDefaultsAndSettings(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingsService(nil, nil)

	state := ReadMaintenanceState(settings)
	if !state.BorderVisible || state.BorderColor != "#dc2626" || state.BorderOpacity != 1 || state.BorderDismissible {
		t.Fatalf("default border state = %#v, want visible red opaque non-dismissible", state)
	}

	settings.Set(ctx, "maintenance_border_visible", false)
	settings.Set(ctx, "maintenance_border_color", "not-a-color")
	settings.Set(ctx, "maintenance_border_opacity", 3)
	settings.Set(ctx, "maintenance_border_dismissible", true)
	state = ReadMaintenanceState(settings)
	if state.BorderVisible || state.BorderColor != "#dc2626" || state.BorderOpacity != 1 || !state.BorderDismissible {
		t.Fatalf("normalized border state = %#v, want hidden default color clamped opacity dismissible", state)
	}

	settings.Set(ctx, "maintenance_border_color", "#123abc")
	settings.Set(ctx, "maintenance_border_opacity", 0.42)
	state = ReadMaintenanceState(settings)
	if state.BorderColor != "#123abc" || state.BorderOpacity != 0.42 {
		t.Fatalf("custom border state = %#v, want configured color and opacity", state)
	}
}
