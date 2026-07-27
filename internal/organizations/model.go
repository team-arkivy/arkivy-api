// Package organizations implements the "Org & Billing" module described in
// docs/ARCHITECTURE.md: organizations, plans, and the Arkivy-side mirror of
// Zitadel users (organization_id, roles). See docs/DATABASE.md §1-2 for the
// original schema this package adapts to MongoDB.
package organizations

import "time"

type PlanCode string

const (
	PlanPersonalFree     PlanCode = "personal_free"
	PlanPersonalPro      PlanCode = "personal_pro"
	PlanPersonalPremium  PlanCode = "personal_premium"
	PlanEnterprise       PlanCode = "enterprise"
	PlanEnterprisePlus   PlanCode = "enterprise_premium"
)

// Plan mirrors DATABASE.md's `plans` table. Pointer fields are nil when the
// dimension is unlimited for that plan (same "NULL = unlimited" convention).
type Plan struct {
	ID                    PlanCode `bson:"_id"                     json:"id"`
	Name                  string   `bson:"name"                    json:"name"`
	MaxSpaces             *int     `bson:"max_spaces"               json:"maxSpaces"`
	MaxPages              *int     `bson:"max_pages"                json:"maxPages"`
	MaxStorageBytes       int64    `bson:"max_storage_bytes"        json:"maxStorageBytes"`
	MaxMembers            int      `bson:"max_members"              json:"maxMembers"`
	MaxGroups             *int     `bson:"max_groups"               json:"maxGroups"`
	GraphViewLevel        string   `bson:"graph_view_level"         json:"graphViewLevel"` // none | per_space | full
	VersionRetentionDays  *int     `bson:"version_retention_days"   json:"versionRetentionDays"`
	MaxGitRepos           int      `bson:"max_git_repos"            json:"maxGitRepos"`
	AllowsIntegrations    bool     `bson:"allows_integrations"      json:"allowsIntegrations"`
	AllowsWhiteLabel      bool     `bson:"allows_white_label"       json:"allowsWhiteLabel"`
}

// Organization mirrors DATABASE.md's `organizations` table.
type Organization struct {
	ID                      string    `bson:"_id"                      json:"id"`
	Name                    string    `bson:"name"                     json:"name"`
	PlanID                  PlanCode  `bson:"plan_id"                  json:"planId"`
	IsPersonal              bool      `bson:"is_personal"              json:"isPersonal"`
	StorageUsedBytes        int64     `bson:"storage_used_bytes"       json:"storageUsedBytes"`
	BrandingLogoURL         string    `bson:"branding_logo_url"        json:"brandingLogoUrl,omitempty"`
	BrandingPrimaryColor    string    `bson:"branding_primary_color"   json:"brandingPrimaryColor,omitempty"`
	BrandingSecondaryColor  string    `bson:"branding_secondary_color" json:"brandingSecondaryColor,omitempty"`
	Status                  string    `bson:"status"                   json:"status"` // active | suspended
	CreatedAt               time.Time `bson:"created_at"               json:"createdAt"`
}

// User is the Arkivy-side mirror of a Zitadel user (DATABASE.md §2), keyed by
// the Zitadel user ID instead of a separately generated UUID — that ID is
// already what every session carries, so reusing it avoids a redundant
// mapping field.
type User struct {
	ID              string    `bson:"_id"               json:"id"` // Zitadel user ID
	OrganizationID  string    `bson:"organization_id"   json:"organizationId"`
	Email           string    `bson:"email"             json:"email"`
	Name            string    `bson:"name"              json:"name"`
	IsPlatformAdmin bool      `bson:"is_platform_admin" json:"isPlatformAdmin"`
	IsSysAdmin      bool      `bson:"is_sys_admin"      json:"isSysAdmin"`
	Status          string    `bson:"status"            json:"status"` // active | inactive
	LastAccessAt    time.Time `bson:"last_access_at"    json:"lastAccessAt,omitempty"`
	CreatedAt       time.Time `bson:"created_at"        json:"createdAt"`
}
