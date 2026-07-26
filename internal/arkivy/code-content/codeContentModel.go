package code_content

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type codeContent struct {
	ID       shared.ID `bson:"id" json:"id"`
	Content  string    `bson:"content" json:"content"`
	Language string    `bson:"language" json:"language"`
	CAt      time.Time `bson:"c_at" json:"cAt"`
	UAt      time.Time `bson:"u_at" json:"uAt"`
}
