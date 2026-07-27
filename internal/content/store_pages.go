package content

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CreatePage creates a Page at the end of its category's order within
// spaceID (RF-DOC-03), fixed at status "draft" / source "manual" until
// Fase 4/RF-ING exist.
func CreatePage(ctx context.Context, db *mongo.Database, orgID, spaceID string, category Category, title, authorID string) (*Page, error) {
	count, err := db.Collection(pagesCollection).CountDocuments(ctx, bson.M{"space_id": spaceID, "category": category})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	p := Page{
		ID:             uuid.New().String(),
		SpaceID:        spaceID,
		OrganizationID: orgID,
		Category:       category,
		OrderIndex:     int(count),
		Title:          title,
		Status:         "draft",
		SourceType:     "manual",
		AuthorID:       authorID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := db.Collection(pagesCollection).InsertOne(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPagesFlat returns every Page of spaceID, unordered by category — used
// internally for cascade deletes.
func ListPagesFlat(ctx context.Context, db *mongo.Database, spaceID string) ([]Page, error) {
	cur, err := db.Collection(pagesCollection).Find(ctx, bson.M{"space_id": spaceID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	pages := []Page{}
	if err := cur.All(ctx, &pages); err != nil {
		return nil, err
	}
	return pages, nil
}

// ListPagesBySpace returns spaceID's pages grouped by category and ordered
// by order_index, for the table of contents (RF-DOC-02).
func ListPagesBySpace(ctx context.Context, db *mongo.Database, spaceID string) ([]TocCategory, error) {
	cur, err := db.Collection(pagesCollection).Find(ctx, bson.M{"space_id": spaceID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	pages := []Page{}
	if err := cur.All(ctx, &pages); err != nil {
		return nil, err
	}

	byCategory := map[Category][]Page{}
	for _, p := range pages {
		byCategory[p.Category] = append(byCategory[p.Category], p)
	}

	order := []Category{CategoryTutorial, CategoryHowTo, CategoryReference, CategoryExplanation, CategoryMultiple}
	toc := make([]TocCategory, 0, len(order))
	for _, cat := range order {
		items := byCategory[cat]
		if items == nil {
			// A category with zero pages must still serialize as "pages": [],
			// not "pages": null — Go's zero-value nil slice would otherwise
			// leak a JSON null that breaks clients expecting an array.
			items = []Page{}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].OrderIndex < items[j].OrderIndex })
		toc = append(toc, TocCategory{Category: cat, Pages: items})
	}
	return toc, nil
}

// GetPage returns a single Page by ID.
func GetPage(ctx context.Context, db *mongo.Database, pageID string) (*Page, error) {
	var p Page
	err := db.Collection(pagesCollection).FindOne(ctx, bson.M{"_id": pageID}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePage updates a Page's title and/or category. Changing category
// appends the page to the end of the new category's order.
func UpdatePage(ctx context.Context, db *mongo.Database, pageID string, title *string, category *Category) error {
	page, err := GetPage(ctx, db, pageID)
	if err != nil {
		return err
	}

	update := bson.M{"updated_at": time.Now()}
	if title != nil {
		update["title"] = *title
	}
	if category != nil && *category != page.Category {
		count, err := db.Collection(pagesCollection).CountDocuments(ctx, bson.M{"space_id": page.SpaceID, "category": *category})
		if err != nil {
			return err
		}
		update["category"] = *category
		update["order_index"] = int(count)
	}

	res, err := db.Collection(pagesCollection).UpdateOne(ctx, bson.M{"_id": pageID}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderPage moves pageID to newIndex within its own category (RF-DOC-03b),
// shifting its siblings and persisting the new order for all of them.
func ReorderPage(ctx context.Context, db *mongo.Database, pageID string, newIndex int) error {
	page, err := GetPage(ctx, db, pageID)
	if err != nil {
		return err
	}

	cur, err := db.Collection(pagesCollection).Find(ctx, bson.M{"space_id": page.SpaceID, "category": page.Category})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	siblings := []Page{}
	if err := cur.All(ctx, &siblings); err != nil {
		return err
	}
	sort.Slice(siblings, func(i, j int) bool { return siblings[i].OrderIndex < siblings[j].OrderIndex })

	from := -1
	for i, s := range siblings {
		if s.ID == pageID {
			from = i
			break
		}
	}
	if from == -1 {
		return ErrNotFound
	}

	moved := siblings[from]
	siblings = append(siblings[:from], siblings[from+1:]...)
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex > len(siblings) {
		newIndex = len(siblings)
	}
	siblings = append(siblings[:newIndex], append([]Page{moved}, siblings[newIndex:]...)...)

	for i, s := range siblings {
		if s.OrderIndex == i {
			continue
		}
		if _, err := db.Collection(pagesCollection).UpdateOne(ctx,
			bson.M{"_id": s.ID},
			bson.M{"$set": bson.M{"order_index": i}},
		); err != nil {
			return err
		}
	}
	return nil
}

// DeletePage removes a Page and cascades to its blocks and their
// attachments (including the underlying GridFS files).
func DeletePage(ctx context.Context, db *mongo.Database, bucket *mongo.GridFSBucket, pageID string) error {
	blocks, err := ListBlocks(ctx, db, pageID)
	if err != nil {
		return err
	}
	for _, b := range blocks {
		attachments, err := ListAttachmentsByBlock(ctx, db, b.ID)
		if err != nil {
			return err
		}
		for _, a := range attachments {
			if err := DeleteAttachment(ctx, db, bucket, a.ID); err != nil {
				return err
			}
		}
	}

	if _, err := db.Collection(blocksCollection).DeleteOne(ctx, bson.M{"_id": pageID}); err != nil {
		return err
	}

	res, err := db.Collection(pagesCollection).DeleteOne(ctx, bson.M{"_id": pageID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}
