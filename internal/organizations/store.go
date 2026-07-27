package organizations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	plansCollection         = "plans"
	organizationsCollection = "organizations"
	usersCollection         = "users"
)

func intPtr(n int) *int { return &n }

// plansSeed is the source of truth for REQUISITOS.md §2.2's plan table.
// Dimensions marked "—" (not available on that plan) are 0, not nil —
// nil means "unlimited" (see Plan in model.go), which is a different thing.
var plansSeed = []Plan{
	{
		ID: PlanPersonalFree, Name: "Personal Free",
		MaxSpaces: intPtr(1), MaxPages: intPtr(50), MaxStorageBytes: 500 * 1024 * 1024,
		MaxMembers: 1, MaxGroups: intPtr(0), GraphViewLevel: "none",
		VersionRetentionDays: intPtr(7), MaxGitRepos: 0,
		AllowsIntegrations: false, AllowsWhiteLabel: false,
	},
	{
		ID: PlanPersonalPro, Name: "Personal Pro",
		MaxSpaces: intPtr(5), MaxPages: intPtr(500), MaxStorageBytes: 5 * 1024 * 1024 * 1024,
		MaxMembers: 4, MaxGroups: intPtr(0), GraphViewLevel: "per_space",
		VersionRetentionDays: intPtr(30), MaxGitRepos: 1,
		AllowsIntegrations: false, AllowsWhiteLabel: false,
	},
	{
		ID: PlanPersonalPremium, Name: "Personal Premium",
		MaxSpaces: intPtr(15), MaxPages: nil, MaxStorageBytes: 20 * 1024 * 1024 * 1024,
		MaxMembers: 11, MaxGroups: intPtr(3), GraphViewLevel: "full",
		VersionRetentionDays: intPtr(90), MaxGitRepos: 5,
		AllowsIntegrations: false, AllowsWhiteLabel: false,
	},
	{
		// max_git_repos no tiene tope explícito en REQUISITOS.md para Enterprise
		// ("Todo + integraciones") — 1000 como valor práctico de "sin límite real".
		ID: PlanEnterprise, Name: "Enterprise",
		MaxSpaces: intPtr(50), MaxPages: nil, MaxStorageBytes: 200 * 1024 * 1024 * 1024,
		MaxMembers: 50, MaxGroups: nil, GraphViewLevel: "full",
		VersionRetentionDays: intPtr(365), MaxGitRepos: 1000,
		AllowsIntegrations: true, AllowsWhiteLabel: false,
	},
	{
		// max_storage_bytes/max_members son la base "negociable" de REQUISITOS.md (1TB+/200+).
		ID: PlanEnterprisePlus, Name: "Enterprise Premium",
		MaxSpaces: nil, MaxPages: nil, MaxStorageBytes: 1024 * 1024 * 1024 * 1024,
		MaxMembers: 200, MaxGroups: nil, GraphViewLevel: "full",
		VersionRetentionDays: nil, MaxGitRepos: 1000,
		AllowsIntegrations: true, AllowsWhiteLabel: true,
	},
}

// SeedPlans upserts the fixed plan catalog. Safe to call on every startup.
func SeedPlans(ctx context.Context, db *mongo.Database) error {
	col := db.Collection(plansCollection)
	for _, p := range plansSeed {
		if _, err := col.ReplaceOne(ctx, bson.M{"_id": p.ID}, p, options.Replace().SetUpsert(true)); err != nil {
			return err
		}
	}
	return nil
}

// EnsureIndexes creates the indexes this package relies on. Safe to call on every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(usersCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "organization_id", Value: 1}, {Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// EnsurePersonalOrganization returns the Arkivy user mirror for zitadelUserID,
// creating a personal Free-plan organization and the user document the first
// time this user is seen (RF-AUTH-02). Safe to call on every login/register —
// a no-op once the user already exists.
func EnsurePersonalOrganization(ctx context.Context, db *mongo.Database, zitadelUserID, email, displayName string) (*User, error) {
	usersCol := db.Collection(usersCollection)

	var existing User
	err := usersCol.FindOne(ctx, bson.M{"_id": zitadelUserID}).Decode(&existing)
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	now := time.Now()
	name := displayName
	if name == "" {
		name = email
	}

	org := Organization{
		ID:               uuid.New().String(),
		Name:             email,
		PlanID:           PlanPersonalFree,
		IsPersonal:       true,
		StorageUsedBytes: 0,
		Status:           "active",
		CreatedAt:        now,
	}
	if _, err := db.Collection(organizationsCollection).InsertOne(ctx, org); err != nil {
		return nil, err
	}

	user := User{
		ID:              zitadelUserID,
		OrganizationID:  org.ID,
		Email:           email,
		Name:            name,
		IsPlatformAdmin: true,
		IsSysAdmin:      false,
		Status:          "active",
		LastAccessAt:    now,
		CreatedAt:       now,
	}
	if _, err := usersCol.InsertOne(ctx, user); err != nil {
		return nil, err
	}

	return &user, nil
}

// IncrementStorageUsed adjusts an organization's storage_used_bytes counter
// by delta (positive on upload, negative on delete). Used by internal/content
// when attachments are uploaded/removed, ahead of Fase 6's plan-limit checks.
func IncrementStorageUsed(ctx context.Context, db *mongo.Database, orgID string, delta int64) error {
	_, err := db.Collection(organizationsCollection).UpdateOne(ctx,
		bson.M{"_id": orgID},
		bson.M{"$inc": bson.M{"storage_used_bytes": delta}},
	)
	return err
}

// ListUsersByOrg returns every user mirrored under orgID, sorted by name —
// used to populate the Groups "add member" picker and other org-directory UI.
func ListUsersByOrg(ctx context.Context, db *mongo.Database, orgID string) ([]User, error) {
	cur, err := db.Collection(usersCollection).Find(ctx,
		bson.M{"organization_id": orgID},
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	users := []User{}
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserWithOrganization loads the Arkivy user mirror and its organization,
// used to enrich GET /auth/me.
func GetUserWithOrganization(ctx context.Context, db *mongo.Database, zitadelUserID string) (*User, *Organization, error) {
	var user User
	if err := db.Collection(usersCollection).FindOne(ctx, bson.M{"_id": zitadelUserID}).Decode(&user); err != nil {
		return nil, nil, err
	}

	var org Organization
	if err := db.Collection(organizationsCollection).FindOne(ctx, bson.M{"_id": user.OrganizationID}).Decode(&org); err != nil {
		return &user, nil, err
	}

	return &user, &org, nil
}
