package groups

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDB connects to the local MongoDB (the same instance the developer runs
// for manual testing) and returns a throwaway database, dropped when the
// test finishes. Skips the test instead of failing if Mongo isn't reachable,
// so this suite never depends on Zitadel or any remote service.
func testDB(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("local MongoDB not reachable: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("local MongoDB not reachable: %v", err)
	}

	db := client.Database("arkivy_groups_test")

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(context.Background())
	})

	return db
}

func TestGroupLifecycleAndCascadeDelete(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	const orgID = "org-1"

	group, err := CreateGroup(ctx, db, orgID, "Lectores")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	list, err := ListGroups(ctx, db, orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListGroups: got %d groups, err=%v", len(list), err)
	}

	if err := RenameGroup(ctx, db, orgID, group.ID, "Editores"); err != nil {
		t.Fatalf("RenameGroup: %v", err)
	}
	got, err := GetGroup(ctx, db, orgID, group.ID)
	if err != nil || got.Name != "Editores" {
		t.Fatalf("GetGroup after rename: %+v, err=%v", got, err)
	}

	// A group is never visible outside its organization.
	if _, err := GetGroup(ctx, db, "other-org", group.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for cross-org access, got %v", err)
	}

	if err := UpsertMember(ctx, db, group.ID, "user-1", AccessReader); err != nil {
		t.Fatalf("UpsertMember: %v", err)
	}
	if err := GrantSpaceAccess(ctx, db, group.ID, "space-1"); err != nil {
		t.Fatalf("GrantSpaceAccess: %v", err)
	}
	if err := GrantPageAccess(ctx, db, group.ID, "page-1"); err != nil {
		t.Fatalf("GrantPageAccess: %v", err)
	}

	if err := DeleteGroup(ctx, db, orgID, group.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, err := GetGroup(ctx, db, orgID, group.ID); err != ErrNotFound {
		t.Fatalf("expected group gone after delete, got %v", err)
	}

	members, _ := ListMembers(ctx, db, group.ID)
	spaces, _ := ListSpaceAccess(ctx, db, group.ID)
	pages, _ := ListPageAccess(ctx, db, group.ID)
	if len(members) != 0 || len(spaces) != 0 || len(pages) != 0 {
		t.Fatalf("expected cascade delete, got members=%d spaces=%d pages=%d", len(members), len(spaces), len(pages))
	}
}

func TestMembership(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	group, err := CreateGroup(ctx, db, "org-1", "Equipo")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	if err := UpsertMember(ctx, db, group.ID, "user-1", AccessReader); err != nil {
		t.Fatalf("UpsertMember user-1: %v", err)
	}
	if err := UpsertMember(ctx, db, group.ID, "user-2", AccessEditor); err != nil {
		t.Fatalf("UpsertMember user-2: %v", err)
	}

	members, err := ListMembers(ctx, db, group.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("ListMembers: got %d, err=%v", len(members), err)
	}

	// Changing a member's role is the same upsert call, not a new membership.
	if err := UpsertMember(ctx, db, group.ID, "user-1", AccessEditor); err != nil {
		t.Fatalf("UpsertMember role change: %v", err)
	}
	members, _ = ListMembers(ctx, db, group.ID)
	if len(members) != 2 {
		t.Fatalf("expected role change not to duplicate membership, got %d members", len(members))
	}
	for _, m := range members {
		if m.UserID == "user-1" && m.Role != AccessEditor {
			t.Fatalf("expected user-1's role to be updated to editor, got %s", m.Role)
		}
	}

	if err := RemoveMember(ctx, db, group.ID, "user-2"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	members, _ = ListMembers(ctx, db, group.ID)
	if len(members) != 1 {
		t.Fatalf("expected 1 member after removal, got %d", len(members))
	}
}

func TestAccessGrantsAreIdempotent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	group, err := CreateGroup(ctx, db, "org-1", "Equipo")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	for range 2 {
		if err := GrantSpaceAccess(ctx, db, group.ID, "space-1"); err != nil {
			t.Fatalf("GrantSpaceAccess: %v", err)
		}
	}
	spaces, err := ListSpaceAccess(ctx, db, group.ID)
	if err != nil || len(spaces) != 1 {
		t.Fatalf("expected 1 space grant after granting twice, got %d, err=%v", len(spaces), err)
	}

	if err := RevokeSpaceAccess(ctx, db, group.ID, "space-1"); err != nil {
		t.Fatalf("RevokeSpaceAccess: %v", err)
	}
	spaces, _ = ListSpaceAccess(ctx, db, group.ID)
	if len(spaces) != 0 {
		t.Fatalf("expected 0 space grants after revoke, got %d", len(spaces))
	}

	for range 2 {
		if err := GrantPageAccess(ctx, db, group.ID, "page-1"); err != nil {
			t.Fatalf("GrantPageAccess: %v", err)
		}
	}
	pages, err := ListPageAccess(ctx, db, group.ID)
	if err != nil || len(pages) != 1 {
		t.Fatalf("expected 1 page grant after granting twice, got %d, err=%v", len(pages), err)
	}
}

func TestEffectiveAccessHighestRoleWins(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	readerGroup, err := CreateGroup(ctx, db, "org-1", "Lectores")
	if err != nil {
		t.Fatalf("CreateGroup readerGroup: %v", err)
	}
	editorGroup, err := CreateGroup(ctx, db, "org-1", "Editores")
	if err != nil {
		t.Fatalf("CreateGroup editorGroup: %v", err)
	}

	if err := UpsertMember(ctx, db, readerGroup.ID, "user-1", AccessReader); err != nil {
		t.Fatalf("UpsertMember reader: %v", err)
	}
	if err := UpsertMember(ctx, db, editorGroup.ID, "user-1", AccessEditor); err != nil {
		t.Fatalf("UpsertMember editor: %v", err)
	}
	if err := GrantSpaceAccess(ctx, db, readerGroup.ID, "space-1"); err != nil {
		t.Fatalf("GrantSpaceAccess readerGroup: %v", err)
	}
	if err := GrantSpaceAccess(ctx, db, editorGroup.ID, "space-1"); err != nil {
		t.Fatalf("GrantSpaceAccess editorGroup: %v", err)
	}

	access, err := EffectiveAccess(ctx, db, "user-1", "space", "space-1")
	if err != nil {
		t.Fatalf("EffectiveAccess: %v", err)
	}
	if access == nil || *access != AccessEditor {
		t.Fatalf("expected editor (highest role wins), got %v", access)
	}

	// No group grants access to this resource.
	access, err = EffectiveAccess(ctx, db, "user-1", "space", "space-2")
	if err != nil {
		t.Fatalf("EffectiveAccess (no access): %v", err)
	}
	if access != nil {
		t.Fatalf("expected nil access, got %v", *access)
	}

	// User isn't a member of any group at all.
	access, err = EffectiveAccess(ctx, db, "stranger", "space", "space-1")
	if err != nil {
		t.Fatalf("EffectiveAccess (non-member): %v", err)
	}
	if access != nil {
		t.Fatalf("expected nil access for non-member, got %v", *access)
	}
}
