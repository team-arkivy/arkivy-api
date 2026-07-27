package content

import (
	"context"

	"arkivy-api/internal/groups"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// meetsLevel reports whether actual satisfies at least min (editor > reader).
func meetsLevel(actual, min groups.AccessLevel) bool {
	if actual == groups.AccessEditor {
		return true
	}
	return min == groups.AccessReader
}

// higher returns whichever of a/b is the higher access level (editor > reader), nil if both nil.
func higher(a, b *groups.AccessLevel) *groups.AccessLevel {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if *a == groups.AccessEditor {
		return a
	}
	return b
}

// SpaceAccess returns isAdmin's user effective access to spaceID: full access
// if isAdmin (Platform Admin bypasses Groups entirely — RF-GRP applies only
// to non-admins), otherwise whatever internal/groups.EffectiveAccess (Fase 2)
// computes from that user's Group memberships. nil means no access — the
// resource is private by default (confirmed for Fase 3).
func SpaceAccess(ctx context.Context, db *mongo.Database, userID string, isAdmin bool, spaceID string) *groups.AccessLevel {
	if isAdmin {
		level := groups.AccessEditor
		return &level
	}
	level, _ := groups.EffectiveAccess(ctx, db, userID, "space", spaceID)
	return level
}

// PageAccess returns a user's effective access to a Page: the higher of its
// own direct grant (group_page_access) and its Space's grant — a Page
// inherits its Space's access, or can have a loose grant of its own
// (docs/DATABASE.md §4).
func PageAccess(ctx context.Context, db *mongo.Database, userID string, isAdmin bool, page Page) *groups.AccessLevel {
	if isAdmin {
		level := groups.AccessEditor
		return &level
	}
	spaceLevel, _ := groups.EffectiveAccess(ctx, db, userID, "space", page.SpaceID)
	pageLevel, _ := groups.EffectiveAccess(ctx, db, userID, "page", page.ID)
	return higher(spaceLevel, pageLevel)
}
