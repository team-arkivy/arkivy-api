package license

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type Status string

const (
	STATUS_ENABLE  Status = "ENABLE"
	STATUS_DISABLE Status = "DISABLE"
)

type License struct {
	ID            shared.ID `bson:"id" json:"id"`
	LicenseType   string    `bson:"license_type" json:"licenseType"`
	Status        string    `bson:"status" json:"status"`
	UserID        shared.ID `bson:"user_Id" json:"userId"`
	PlatformID    shared.ID `bson:"platform_Id" json:"platformId"`
	UserCreatorId shared.ID `bson:"userCreator_Id" json:"userCreatorId"`
	CAt           time.Time `bson:"c_at" json:"cAt"`
	UAt           time.Time `bson:"u_at" json:"uAt"`
}
