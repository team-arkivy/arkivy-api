package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func validBlockType(t BlockType) bool {
	switch t {
	case BlockText, BlockCode, BlockFile, BlockLink:
		return true
	default:
		return false
	}
}

// ListBlocks returns pageID's blocks in order. A page with no blocks
// document yet (never saved) returns an empty slice, not an error.
func ListBlocks(ctx context.Context, db *mongo.Database, pageID string) ([]Block, error) {
	var doc pageBlocks
	err := db.Collection(blocksCollection).FindOne(ctx, bson.M{"_id": pageID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return []Block{}, nil
	}
	if err != nil {
		return nil, err
	}
	if doc.Blocks == nil {
		return []Block{}, nil
	}
	return doc.Blocks, nil
}

// FindPageIDByBlockID looks up which page owns a given block ID — needed
// because attachment routes are addressed by block, not by page.
func FindPageIDByBlockID(ctx context.Context, db *mongo.Database, blockID string) (string, error) {
	var doc pageBlocks
	err := db.Collection(blocksCollection).FindOne(ctx,
		bson.M{"blocks.id": blockID},
		options.FindOne().SetProjection(bson.M{"_id": 1}),
	).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return doc.PageID, nil
}

// ReplaceBlocks validates and persists pageID's full ordered block list in
// one call (RF-DOC-04/05) — the natural backend counterpart of how the
// frontend already treats a page's blocks as one in-memory array. Assigns an
// ID to any block missing one and normalizes OrderIndex to array position.
func ReplaceBlocks(ctx context.Context, db *mongo.Database, pageID string, blocks []Block) error {
	for i := range blocks {
		if !validBlockType(blocks[i].Type) {
			return fmt.Errorf("invalid block type %q at index %d", blocks[i].Type, i)
		}
		if blocks[i].ID == "" {
			blocks[i].ID = uuid.New().String()
		}
		blocks[i].OrderIndex = i
	}

	_, err := db.Collection(blocksCollection).UpdateOne(ctx,
		bson.M{"_id": pageID},
		bson.M{"$set": bson.M{"_id": pageID, "blocks": blocks}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}
