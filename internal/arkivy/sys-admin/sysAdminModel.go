package sys_admin

import (
	"arkivy-api/internal/arkivy/shared"
	"time"
)

type SysAdmin struct {
	ID         shared.ID `bson:"_id" json:"id"`
	LastAccess time.Time `bson:"lastAccess" json:"lastAccess"`
	UserId     shared.ID `bson:"userId" json:"userId"`
	CAt        time.Time `bson:"c_at" json:"cAt"`
}
