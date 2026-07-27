package content

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	spacesCollection      = "spaces"
	pagesCollection       = "pages"
	blocksCollection      = "blocks"
	attachmentsCollection = "attachments"
)

var ErrNotFound = errors.New("not found")

// EnsureIndexes creates the indexes this package relies on. Safe to call on
// every startup.
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection(spacesCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "organization_id", Value: 1}},
	}); err != nil {
		return err
	}

	if _, err := db.Collection(pagesCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "space_id", Value: 1}, {Key: "category", Value: 1}, {Key: "order_index", Value: 1}},
	}); err != nil {
		return err
	}

	_, err := db.Collection(attachmentsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "block_id", Value: 1}},
		Options: options.Index(),
	})
	return err
}
