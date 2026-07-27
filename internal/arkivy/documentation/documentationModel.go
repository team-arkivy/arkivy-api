package documentation

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type documentationModel struct {
	ID            shared.ID `bson:"id" json:"id"`
	Name          string    `bson:"name" json:"name"`
	ContentIdList []string  `bson:"content_id_list" json:"contentIdList"`
	CAt           time.Time `bson:"c_at" json:"cAt"`
	UAt           time.Time `bson:"u_at" json:"uAt"`
}
