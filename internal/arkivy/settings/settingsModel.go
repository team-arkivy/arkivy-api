package settings

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type Settings struct {
	ID                  shared.ID `bson:"id" json:"id"`
	PrimaryColorLight   string    `bson:"primary_color_light" json:"primaryColorLight"`
	PrimaryColorDark    string    `bson:"primary_color_dark" json:"primaryColorDark"`
	SecondaryColorLight string    `bson:"secondary_color_light" json:"secondaryColorLight"`
	SecondaryColorDark  string    `bson:"secondary_color_dark" json:"secondaryColorDark"`
	AccentColorLight    string    `bson:"accent_color_light" json:"accentColorLight"`
	AccentColorDark     string    `bson:"accent_color_dark" json:"accentColorDark"`
	LogoUrlLight        string    `bson:"logo_url_light" json:"logoUrlLight"`
	LogoUrlDark         string    `bson:"logo_url_dark" json:"logoUrlDark"`
	CAt                 time.Time `bson:"c_at" json:"cAt"`
	UAt                 time.Time `bson:"u_at" json:"uAt"`
}
