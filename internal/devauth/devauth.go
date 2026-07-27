// Package devauth implements a local-only authentication bypass for working
// on the real backend/MongoDB without Zitadel credentials. It only takes
// effect when config.DevAuthBypass is true — see
// internal/middleware.SessionMiddleware (skips the real Zitadel session
// check) and cmd/main.go (seeds the fixed user below on startup). Never set
// DEV_AUTH_BYPASS=true outside a developer's own machine.
package devauth

import (
	"context"

	"arkivy-api/internal/organizations"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	// UserID is the fixed Arkivy-side user mirror ID used for every request
	// while the bypass is active — it plays the role a real Zitadel user ID
	// would.
	UserID      = "dev-user-1"
	Email       = "dev@arkivy.local"
	DisplayName = "Dev User"

	// SessionID/SessionToken are the fixed values POST /auth/dev-login
	// returns; SessionMiddleware doesn't actually check them against
	// anything when the bypass is on, but the frontend still stores and
	// sends them like a real session.
	SessionID    = "dev-session-id"
	SessionToken = "dev-session-token"
)

// SeedUser ensures the fixed dev user (and its personal organization) exists,
// mirroring what a real first login/register does via RF-AUTH-02. Safe to
// call on every startup.
func SeedUser(ctx context.Context, db *mongo.Database) error {
	_, err := organizations.EnsurePersonalOrganization(ctx, db, UserID, Email, DisplayName)
	return err
}
