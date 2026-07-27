package groups

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
	groupsCollection      = "groups"
	membersCollection     = "group_members"
	spaceAccessCollection = "group_space_access"
	pageAccessCollection  = "group_page_access"
)

var ErrNotFound = errors.New("not found")

// EnsureIndexes creates the indexes this package relies on. Safe to call on
// every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection(groupsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "organization_id", Value: 1}},
	}); err != nil {
		return err
	}

	if _, err := db.Collection(membersCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if _, err := db.Collection(spaceAccessCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "space_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	_, err := db.Collection(pageAccessCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "page_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// CreateGroup creates a new Group in the given organization (RF-GRP-01).
func CreateGroup(ctx context.Context, db *mongo.Database, orgID, name string) (*Group, error) {
	g := Group{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Name:           name,
		CreatedAt:      time.Now(),
	}
	if _, err := db.Collection(groupsCollection).InsertOne(ctx, g); err != nil {
		return nil, err
	}
	return &g, nil
}

// ListGroups returns every Group that belongs to orgID.
func ListGroups(ctx context.Context, db *mongo.Database, orgID string) ([]Group, error) {
	cur, err := db.Collection(groupsCollection).Find(ctx, bson.M{"organization_id": orgID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	groups := []Group{}
	if err := cur.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// GetGroup returns a Group scoped to orgID — a group is never visible
// outside the organization that owns it.
func GetGroup(ctx context.Context, db *mongo.Database, orgID, groupID string) (*Group, error) {
	var g Group
	err := db.Collection(groupsCollection).FindOne(ctx, bson.M{"_id": groupID, "organization_id": orgID}).Decode(&g)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetGroupDetail loads a Group together with its members and access grants,
// used by GET /groups/:id.
func GetGroupDetail(ctx context.Context, db *mongo.Database, orgID, groupID string) (*GroupDetail, error) {
	g, err := GetGroup(ctx, db, orgID, groupID)
	if err != nil {
		return nil, err
	}

	members, err := ListMembers(ctx, db, groupID)
	if err != nil {
		return nil, err
	}
	spaces, err := ListSpaceAccess(ctx, db, groupID)
	if err != nil {
		return nil, err
	}
	pages, err := ListPageAccess(ctx, db, groupID)
	if err != nil {
		return nil, err
	}

	return &GroupDetail{Group: *g, Members: members, Spaces: spaces, Pages: pages}, nil
}

// RenameGroup updates a Group's name, scoped to orgID.
func RenameGroup(ctx context.Context, db *mongo.Database, orgID, groupID, name string) error {
	res, err := db.Collection(groupsCollection).UpdateOne(ctx,
		bson.M{"_id": groupID, "organization_id": orgID},
		bson.M{"$set": bson.M{"name": name}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteGroup removes a Group and cascades the delete to its members and
// access grants (RF-GRP-06).
func DeleteGroup(ctx context.Context, db *mongo.Database, orgID, groupID string) error {
	res, err := db.Collection(groupsCollection).DeleteOne(ctx, bson.M{"_id": groupID, "organization_id": orgID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}

	if _, err := db.Collection(membersCollection).DeleteMany(ctx, bson.M{"group_id": groupID}); err != nil {
		return err
	}
	if _, err := db.Collection(spaceAccessCollection).DeleteMany(ctx, bson.M{"group_id": groupID}); err != nil {
		return err
	}
	if _, err := db.Collection(pageAccessCollection).DeleteMany(ctx, bson.M{"group_id": groupID}); err != nil {
		return err
	}
	return nil
}

// UpsertMember adds userID to groupID with the given role, or updates the
// role if the membership already exists.
func UpsertMember(ctx context.Context, db *mongo.Database, groupID, userID string, role AccessLevel) error {
	_, err := db.Collection(membersCollection).UpdateOne(ctx,
		bson.M{"group_id": groupID, "user_id": userID},
		bson.M{"$set": bson.M{"group_id": groupID, "user_id": userID, "role": role}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// RemoveMember removes userID from groupID (RF-GRP-06).
func RemoveMember(ctx context.Context, db *mongo.Database, groupID, userID string) error {
	_, err := db.Collection(membersCollection).DeleteOne(ctx, bson.M{"group_id": groupID, "user_id": userID})
	return err
}

// ListMembers returns every Member of groupID.
func ListMembers(ctx context.Context, db *mongo.Database, groupID string) ([]Member, error) {
	cur, err := db.Collection(membersCollection).Find(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	members := []Member{}
	if err := cur.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// GrantSpaceAccess gives groupID access to spaceID (RF-GRP-02). Idempotent.
func GrantSpaceAccess(ctx context.Context, db *mongo.Database, groupID, spaceID string) error {
	_, err := db.Collection(spaceAccessCollection).UpdateOne(ctx,
		bson.M{"group_id": groupID, "space_id": spaceID},
		bson.M{"$set": bson.M{"group_id": groupID, "space_id": spaceID}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// RevokeSpaceAccess removes groupID's access to spaceID (RF-GRP-06).
func RevokeSpaceAccess(ctx context.Context, db *mongo.Database, groupID, spaceID string) error {
	_, err := db.Collection(spaceAccessCollection).DeleteOne(ctx, bson.M{"group_id": groupID, "space_id": spaceID})
	return err
}

// ListSpaceAccess returns every Space groupID has been granted access to.
func ListSpaceAccess(ctx context.Context, db *mongo.Database, groupID string) ([]SpaceAccess, error) {
	cur, err := db.Collection(spaceAccessCollection).Find(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	grants := []SpaceAccess{}
	if err := cur.All(ctx, &grants); err != nil {
		return nil, err
	}
	return grants, nil
}

// GrantPageAccess gives groupID access to a loose pageID (RF-GRP-02). Idempotent.
func GrantPageAccess(ctx context.Context, db *mongo.Database, groupID, pageID string) error {
	_, err := db.Collection(pageAccessCollection).UpdateOne(ctx,
		bson.M{"group_id": groupID, "page_id": pageID},
		bson.M{"$set": bson.M{"group_id": groupID, "page_id": pageID}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

// RevokePageAccess removes groupID's access to pageID (RF-GRP-06).
func RevokePageAccess(ctx context.Context, db *mongo.Database, groupID, pageID string) error {
	_, err := db.Collection(pageAccessCollection).DeleteOne(ctx, bson.M{"group_id": groupID, "page_id": pageID})
	return err
}

// ListPageAccess returns every loose Page groupID has been granted access to.
func ListPageAccess(ctx context.Context, db *mongo.Database, groupID string) ([]PageAccess, error) {
	cur, err := db.Collection(pageAccessCollection).Find(ctx, bson.M{"group_id": groupID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	grants := []PageAccess{}
	if err := cur.All(ctx, &grants); err != nil {
		return nil, err
	}
	return grants, nil
}

// EffectiveAccess computes a user's effective access level on a resource
// (RF-GRP-04): the highest role (editor > reader) among every Group the user
// is a member of that has been granted access to that resource. Returns nil
// if no such Group exists. resourceType is "space" or "page".
func EffectiveAccess(ctx context.Context, db *mongo.Database, userID, resourceType, resourceID string) (*AccessLevel, error) {
	memberCur, err := db.Collection(membersCollection).Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer memberCur.Close(ctx)

	memberships := []Member{}
	if err := memberCur.All(ctx, &memberships); err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, nil
	}

	roleByGroup := make(map[string]AccessLevel, len(memberships))
	groupIDs := make([]string, 0, len(memberships))
	for _, m := range memberships {
		roleByGroup[m.GroupID] = m.Role
		groupIDs = append(groupIDs, m.GroupID)
	}

	accessCollection := spaceAccessCollection
	resourceField := "space_id"
	if resourceType == "page" {
		accessCollection = pageAccessCollection
		resourceField = "page_id"
	}

	accessCur, err := db.Collection(accessCollection).Find(ctx, bson.M{
		"group_id":    bson.M{"$in": groupIDs},
		resourceField: resourceID,
	})
	if err != nil {
		return nil, err
	}
	defer accessCur.Close(ctx)

	var best *AccessLevel
	for accessCur.Next(ctx) {
		var raw bson.M
		if err := accessCur.Decode(&raw); err != nil {
			return nil, err
		}
		groupID, _ := raw["group_id"].(string)
		role, ok := roleByGroup[groupID]
		if !ok {
			continue
		}
		if best == nil || role.higherThan(*best) {
			r := role
			best = &r
		}
	}
	if err := accessCur.Err(); err != nil {
		return nil, err
	}

	return best, nil
}
