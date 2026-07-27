package user_profile

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type UserProfile struct {
	ID           shared.ID `bson:"_id" json:"id"`
	FirstName    string    `bson:"firstName" json:"firstName"`
	LastName     string    `bson:"lastName" json:"lastName"`
	EmailAddress string    `bson:"emailAddress" json:"emailAddress"`
	Status       string    `bson:"status" json:"status"`
	LastAccess   time.Time `bson:"lastAccess" json:"lastAccess"`
	CAt          time.Time `bson:"c_at" json:"cAt"`
	UAt          time.Time `bson:"u_at" json:"uAt"`
}
