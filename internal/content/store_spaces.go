package content

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CreateSpace creates a new Space in the given organization (RF-DOC-01).
func CreateSpace(ctx context.Context, db *mongo.Database, orgID, name string) (*Space, error) {
	s := Space{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Name:           name,
		CreatedAt:      time.Now(),
	}
	if _, err := db.Collection(spacesCollection).InsertOne(ctx, s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSpacesByOrg returns every Space that belongs to orgID, regardless of
// per-user access — callers filter by effective access themselves (see
// access.go), since Platform Admins can see everything.
func ListSpacesByOrg(ctx context.Context, db *mongo.Database, orgID string) ([]Space, error) {
	cur, err := db.Collection(spacesCollection).Find(ctx, bson.M{"organization_id": orgID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	spaces := []Space{}
	if err := cur.All(ctx, &spaces); err != nil {
		return nil, err
	}
	return spaces, nil
}

// GetSpace returns a Space scoped to orgID.
func GetSpace(ctx context.Context, db *mongo.Database, orgID, spaceID string) (*Space, error) {
	var s Space
	err := db.Collection(spacesCollection).FindOne(ctx, bson.M{"_id": spaceID, "organization_id": orgID}).Decode(&s)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// RenameSpace updates a Space's name, scoped to orgID.
func RenameSpace(ctx context.Context, db *mongo.Database, orgID, spaceID, name string) error {
	res, err := db.Collection(spacesCollection).UpdateOne(ctx,
		bson.M{"_id": spaceID, "organization_id": orgID},
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

// DeleteSpace removes a Space and cascades the delete to its pages, their
// blocks, and any attachments on those blocks (GridFS files included).
func DeleteSpace(ctx context.Context, db *mongo.Database, bucket *mongo.GridFSBucket, orgID, spaceID string) error {
	res, err := db.Collection(spacesCollection).DeleteOne(ctx, bson.M{"_id": spaceID, "organization_id": orgID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}

	pages, err := ListPagesFlat(ctx, db, spaceID)
	if err != nil {
		return err
	}
	for _, p := range pages {
		if err := DeletePage(ctx, db, bucket, p.ID); err != nil {
			return err
		}
	}
	return nil
}
