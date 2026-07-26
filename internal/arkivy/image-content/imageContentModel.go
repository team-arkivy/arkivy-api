package image_content

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type ImageContent struct {
	ID    shared.ID `bson:"id" json:"id"`
	Url   string    `bson:"url" json:"url"`
	Width int       `bson:"width" json:"width"`
	CAt   time.Time `bson:"c_at" json:"cAt"`
	UAt   time.Time `bson:"u_at" json:"uAt"`
}
