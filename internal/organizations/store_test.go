package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDB connects to the developer's local native mongod and returns a
// disposable database, dropped at the end of the test. Mirrors the pattern
// in internal/groups and internal/content — no Zitadel involved.
func testDB(t *testing.T) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("local mongod not reachable, skipping:", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("local mongod not reachable, skipping:", err)
	}

	dbName := "arkivy_organizations_test"
	db := client.Database(dbName)

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
	})

	return db
}

func insertTestUser(t *testing.T, db *mongo.Database, orgID, name, email string) User {
	t.Helper()
	u := User{
		ID:              uuid.New().String(),
		OrganizationID:  orgID,
		Email:           email,
		Name:            name,
		IsPlatformAdmin: false,
		Status:          "active",
		CreatedAt:       time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := db.Collection(usersCollection).InsertOne(ctx, u); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return u
}

func TestListUsersByOrg(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	orgA := uuid.New().String()
	orgB := uuid.New().String()

	insertTestUser(t, db, orgA, "Carlos", "carlos@example.com")
	insertTestUser(t, db, orgA, "Ana", "ana@example.com")
	insertTestUser(t, db, orgB, "Beatriz", "beatriz@example.com")

	users, err := ListUsersByOrg(ctx, db, orgA)
	if err != nil {
		t.Fatalf("ListUsersByOrg: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users in orgA, got %d", len(users))
	}
	if users[0].Name != "Ana" || users[1].Name != "Carlos" {
		t.Fatalf("expected users sorted by name (Ana, Carlos), got (%s, %s)", users[0].Name, users[1].Name)
	}

	usersB, err := ListUsersByOrg(ctx, db, orgB)
	if err != nil {
		t.Fatalf("ListUsersByOrg orgB: %v", err)
	}
	if len(usersB) != 1 || usersB[0].Name != "Beatriz" {
		t.Fatalf("expected only Beatriz in orgB, got %+v", usersB)
	}

	empty, err := ListUsersByOrg(ctx, db, uuid.New().String())
	if err != nil {
		t.Fatalf("ListUsersByOrg empty org: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no users for unknown org, got %d", len(empty))
	}
}
