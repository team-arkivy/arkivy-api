package content

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// SearchPages finds pages within orgID whose title or block content matches
// query (RF-DOC-07), case-insensitive, deduplicated.
func SearchPages(ctx context.Context, db *mongo.Database, orgID, query string) ([]Page, error) {
	if query == "" {
		return []Page{}, nil
	}
	regex := bson.M{"$regex": query, "$options": "i"}

	titleCur, err := db.Collection(pagesCollection).Find(ctx, bson.M{
		"organization_id": orgID,
		"title":           regex,
	})
	if err != nil {
		return nil, err
	}
	defer titleCur.Close(ctx)

	byID := map[string]Page{}
	titleMatches := []Page{}
	if err := titleCur.All(ctx, &titleMatches); err != nil {
		return nil, err
	}
	for _, p := range titleMatches {
		byID[p.ID] = p
	}

	blockCur, err := db.Collection(blocksCollection).Find(ctx, bson.M{
		"blocks": bson.M{"$elemMatch": bson.M{"content": regex}},
	})
	if err != nil {
		return nil, err
	}
	defer blockCur.Close(ctx)

	var blockDocs []pageBlocks
	if err := blockCur.All(ctx, &blockDocs); err != nil {
		return nil, err
	}

	matchedPageIDs := make([]string, 0, len(blockDocs))
	for _, doc := range blockDocs {
		if _, already := byID[doc.PageID]; !already {
			matchedPageIDs = append(matchedPageIDs, doc.PageID)
		}
	}

	if len(matchedPageIDs) > 0 {
		pageCur, err := db.Collection(pagesCollection).Find(ctx, bson.M{
			"organization_id": orgID,
			"_id":             bson.M{"$in": matchedPageIDs},
		})
		if err != nil {
			return nil, err
		}
		defer pageCur.Close(ctx)

		var pages []Page
		if err := pageCur.All(ctx, &pages); err != nil {
			return nil, err
		}
		for _, p := range pages {
			byID[p.ID] = p
		}
	}

	results := make([]Page, 0, len(byID))
	for _, p := range byID {
		results = append(results, p)
	}
	return results, nil
}
