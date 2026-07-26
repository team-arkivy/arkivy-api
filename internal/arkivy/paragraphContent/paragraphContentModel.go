package paragraphContent

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type ParagraphContent struct {
	ID       shared.ID `bson:"id" json:"id"`
	Content  string    `bson:"content" json:"content"`
	FontSize int       `bson:"font_size" json:"fontSize"`
	FontName string    `bson:"font_name" json:"fontName"`
	CAt      time.Time `bson:"c_at" json:"cAt"`
	UAt      time.Time `bson:"u_at" json:"uAt"`
}
