package section_doc

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type sectionDoc struct {
	ID          shared.ID `bson:"_id" json:"id"`
	PlatformId  shared.ID `bson:"platform_id" json:"platformId"`
	Name        string    `bson:"name" json:"name"`
	FilesIdList []string  `bson:"files_id_list" json:"filesIdList"`
	CAt         time.Time `bson:"c_at" json:"cAt"`
	UAt         time.Time `bson:"u_at" json:"uAt"`
}
