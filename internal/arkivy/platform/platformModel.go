package platform

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type Platform struct {
	ID         shared.ID `bson:"id" json:"id"`
	Name       string    `bson:"name" json:"name"`
	SettingsId shared.ID `bson:"settings_id" json:"settingsId"`
	CAt        time.Time `bson:"c_at" json:"cAt"`
}
