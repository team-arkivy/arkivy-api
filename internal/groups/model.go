// Package groups implements Groups & RBAC (docs/REQUISITOS.md §10, RF-GRP)
// on top of the Mongo schema sketched in docs/DATABASE.md §4. A Group's
// Members each hold a role (editor/reader); that role is the only access
// level that exists — it applies uniformly to every Space/Page the Group has
// been granted access to (RF-GRP-03). There is no per-resource access level.
package groups

import "time"

type AccessLevel string

const (
	AccessEditor AccessLevel = "editor"
	AccessReader AccessLevel = "reader"
)

// higherThan reports whether a is a strictly higher access level than b.
func (a AccessLevel) higherThan(b AccessLevel) bool {
	return a == AccessEditor && b == AccessReader
}

// Group mirrors DATABASE.md's `groups` table.
type Group struct {
	ID             string    `bson:"_id"             json:"id"`
	OrganizationID string    `bson:"organization_id" json:"organizationId"`
	Name           string    `bson:"name"             json:"name"`
	CreatedAt      time.Time `bson:"created_at"       json:"createdAt"`
}

// Member mirrors DATABASE.md's `group_members` table — the single source of
// a user's access level within a Group.
type Member struct {
	GroupID string      `bson:"group_id" json:"groupId"`
	UserID  string      `bson:"user_id"  json:"userId"`
	Role    AccessLevel `bson:"role"     json:"role"`
}

// SpaceAccess mirrors DATABASE.md's `group_space_access` table: a Group's
// access to a full Space. Space itself doesn't exist as a collection yet
// (Fase 3), so SpaceID is an opaque reference for now.
type SpaceAccess struct {
	GroupID string `bson:"group_id" json:"groupId"`
	SpaceID string `bson:"space_id" json:"spaceId"`
}

// PageAccess mirrors DATABASE.md's `group_page_access` table: a Group's
// access to a loose Page outside its Space's access.
type PageAccess struct {
	GroupID string `bson:"group_id" json:"groupId"`
	PageID  string `bson:"page_id"  json:"pageId"`
}

// GroupDetail bundles a Group with its members and access grants, used by
// GET /groups/:id.
type GroupDetail struct {
	Group   Group         `json:"group"`
	Members []Member      `json:"members"`
	Spaces  []SpaceAccess `json:"spaces"`
	Pages   []PageAccess  `json:"pages"`
}
