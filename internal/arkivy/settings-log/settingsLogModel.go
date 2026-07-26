package settings_log

import (
	"arkivy-api/internal/arkivy/settings"
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type settingsLog struct {
	ID                shared.ID         `bson:"id" json:"id"`
	PlatformId        shared.ID         `bson:"platform_id" json:"platformId"`
	SettingsId        shared.ID         `bson:"settings_id" json:"settingsId"`
	ModifierLicenseId shared.ID         `bson:"modifier_license_id" json:"modifierLicenseId"`
	OldColors         settings.Settings `bson:"old_colors" json:"oldColors"`
	NewColors         settings.Settings `bson:"new_colors" json:"newColors"`
	CAt               time.Time         `bson:"c_at" json:"cAt"`
}
