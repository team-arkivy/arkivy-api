package group

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type Status string

const (
	STATUS_ENABLE  Status = "ENABLE"
	STATUS_DISABLE Status = "DISABLE"
)

type Group struct {
	ID           shared.ID `bson:"id" json:"id"`
	Name         string    `bson:"name" json:"name"`
	EmailAddress string    `bson:"email_address" json:"emailAddress"`
	Status       Status    `bson:"status" json:"status"`
	PlatformId   shared.ID `bson:"platform_id" json:"platformId"`
	CAt          time.Time `bson:"c_at" json:"cAt"`
	UAt          time.Time `bson:"u_at" json:"uAt"`
}
