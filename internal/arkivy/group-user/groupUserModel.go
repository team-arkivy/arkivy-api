package group_user

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type GroupUser struct {
	ID      shared.ID `bson:"_id" json:"id"`
	UserId  shared.ID `bson:"user_id" json:"userId"`
	GroupId string    `bson:"group_id" json:"groupId"`
	CAt     time.Time `bson:"c_at" json:"CAt"`
}
