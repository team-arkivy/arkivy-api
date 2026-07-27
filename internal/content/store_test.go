package content

import (
	"context"
	"strings"
	"testing"
	"time"

	"arkivy-api/internal/groups"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// testDB connects to the local MongoDB and returns a throwaway database,
// dropped when the test finishes. Skips instead of failing if Mongo isn't
// reachable — this suite needs no Zitadel and no remote service.
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

	db := client.Database("arkivy_content_test")

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(context.Background())
	})

	return db
}

func TestSpacePageCascadeAndReorder(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	bucket := db.GridFSBucket()
	const orgID = "org-1"

	space, err := CreateSpace(ctx, db, orgID, "Docs")
	if err != nil {
		t.Fatalf("CreateSpace: %v", err)
	}

	p1, err := CreatePage(ctx, db, orgID, space.ID, CategoryTutorial, "First", "user-1")
	if err != nil {
		t.Fatalf("CreatePage p1: %v", err)
	}
	p2, err := CreatePage(ctx, db, orgID, space.ID, CategoryTutorial, "Second", "user-1")
	if err != nil {
		t.Fatalf("CreatePage p2: %v", err)
	}
	p3, err := CreatePage(ctx, db, orgID, space.ID, CategoryTutorial, "Third", "user-1")
	if err != nil {
		t.Fatalf("CreatePage p3: %v", err)
	}
	if p1.OrderIndex != 0 || p2.OrderIndex != 1 || p3.OrderIndex != 2 {
		t.Fatalf("expected sequential order_index, got %d %d %d", p1.OrderIndex, p2.OrderIndex, p3.OrderIndex)
	}

	if err := ReorderPage(ctx, db, p3.ID, 0); err != nil {
		t.Fatalf("ReorderPage: %v", err)
	}
	toc, err := ListPagesBySpace(ctx, db, space.ID)
	if err != nil {
		t.Fatalf("ListPagesBySpace: %v", err)
	}
	var tutorialPages []Page
	for _, tc := range toc {
		if tc.Category == CategoryTutorial {
			tutorialPages = tc.Pages
		}
	}
	if len(tutorialPages) != 3 || tutorialPages[0].ID != p3.ID || tutorialPages[1].ID != p1.ID || tutorialPages[2].ID != p2.ID {
		t.Fatalf("expected order [p3,p1,p2], got %+v", tutorialPages)
	}

	if err := ReplaceBlocks(ctx, db, p1.ID, []Block{{Type: BlockFile}}); err != nil {
		t.Fatalf("ReplaceBlocks: %v", err)
	}
	blocks, _ := ListBlocks(ctx, db, p1.ID)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block on p1, got %d", len(blocks))
	}
	insertOrg(t, db, orgID)
	if _, err := UploadAttachment(ctx, db, bucket, orgID, blocks[0].ID, "notes.txt", 5, "user-1", strings.NewReader("hello")); err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}

	if err := DeleteSpace(ctx, db, bucket, orgID, space.ID); err != nil {
		t.Fatalf("DeleteSpace: %v", err)
	}
	if _, err := GetSpace(ctx, db, orgID, space.ID); err != ErrNotFound {
		t.Fatalf("expected space gone, got %v", err)
	}
	if _, err := GetPage(ctx, db, p1.ID); err != ErrNotFound {
		t.Fatalf("expected page gone, got %v", err)
	}
	remaining, _ := ListBlocks(ctx, db, p1.ID)
	if len(remaining) != 0 {
		t.Fatalf("expected blocks gone after cascade delete, got %d", len(remaining))
	}
	attachments, _ := ListAttachmentsByBlock(ctx, db, blocks[0].ID)
	if len(attachments) != 0 {
		t.Fatalf("expected attachments gone after cascade delete, got %d", len(attachments))
	}
}

func TestReplaceBlocksTypedContent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	space, _ := CreateSpace(ctx, db, "org-1", "Docs")
	page, err := CreatePage(ctx, db, "org-1", space.ID, CategoryReference, "API", "user-1")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}

	lang := "go"
	input := []Block{
		{Type: BlockText, Content: "hello"},
		{Type: BlockCode, Content: "fmt.Println()", Language: &lang},
		{Type: BlockFile, Content: ""},
		{Type: BlockLink, Content: "", Metadata: map[string]any{"target_page_id": "abc"}},
	}
	if err := ReplaceBlocks(ctx, db, page.ID, input); err != nil {
		t.Fatalf("ReplaceBlocks: %v", err)
	}

	got, err := ListBlocks(ctx, db, page.ID)
	if err != nil || len(got) != 4 {
		t.Fatalf("ListBlocks: got %d blocks, err=%v", len(got), err)
	}
	for i, b := range got {
		if b.ID == "" {
			t.Fatalf("block %d missing generated ID", i)
		}
		if b.OrderIndex != i {
			t.Fatalf("block %d has order_index %d, want %d", i, b.OrderIndex, i)
		}
	}
	if got[1].Language == nil || *got[1].Language != "go" {
		t.Fatalf("expected code block language 'go', got %v", got[1].Language)
	}
	if got[3].Metadata["target_page_id"] != "abc" {
		t.Fatalf("expected link block metadata target_page_id=abc, got %v", got[3].Metadata)
	}

	if err := ReplaceBlocks(ctx, db, page.ID, []Block{{Type: "bogus"}}); err == nil {
		t.Fatalf("expected error for invalid block type")
	}
}

func TestAttachmentsValidationAndStorageAccounting(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	bucket := db.GridFSBucket()
	const orgID = "org-1"
	insertOrg(t, db, orgID)

	space, _ := CreateSpace(ctx, db, orgID, "Docs")
	page, _ := CreatePage(ctx, db, orgID, space.ID, CategoryReference, "Files", "user-1")
	if err := ReplaceBlocks(ctx, db, page.ID, []Block{{Type: BlockFile}}); err != nil {
		t.Fatalf("ReplaceBlocks: %v", err)
	}
	blocks, _ := ListBlocks(ctx, db, page.ID)
	blockID := blocks[0].ID

	body := "hello world"
	att, err := UploadAttachment(ctx, db, bucket, orgID, blockID, "notes.txt", int64(len(body)), "user-1", strings.NewReader(body))
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	if att.FileType != FileTxt {
		t.Fatalf("expected FileTxt, got %s", att.FileType)
	}
	if got := storageUsed(t, db, orgID); got != int64(len(body)) {
		t.Fatalf("expected storage_used_bytes=%d after upload, got %d", len(body), got)
	}

	if _, err := UploadAttachment(ctx, db, bucket, orgID, blockID, "virus.exe", 10, "user-1", strings.NewReader("x")); err != ErrUnsupportedFileType {
		t.Fatalf("expected ErrUnsupportedFileType, got %v", err)
	}
	if _, err := UploadAttachment(ctx, db, bucket, orgID, blockID, "big.pdf", MaxAttachmentBytes+1, "user-1", strings.NewReader("x")); err != ErrFileTooLarge {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}

	if err := DeleteAttachment(ctx, db, bucket, att.ID); err != nil {
		t.Fatalf("DeleteAttachment: %v", err)
	}
	if got := storageUsed(t, db, orgID); got != 0 {
		t.Fatalf("expected storage_used_bytes=0 after delete, got %d", got)
	}
	if _, _, err := OpenAttachmentDownload(ctx, db, bucket, att.ID); err != ErrNotFound {
		t.Fatalf("expected attachment gone after delete, got %v", err)
	}
}

func TestSearchPagesByTitleAndContent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	const orgID = "org-1"

	space, _ := CreateSpace(ctx, db, orgID, "Docs")
	titleMatch, err := CreatePage(ctx, db, orgID, space.ID, CategoryTutorial, "Guía de instalación", "user-1")
	if err != nil {
		t.Fatalf("CreatePage titleMatch: %v", err)
	}
	contentMatch, err := CreatePage(ctx, db, orgID, space.ID, CategoryTutorial, "Otro documento", "user-1")
	if err != nil {
		t.Fatalf("CreatePage contentMatch: %v", err)
	}
	if err := ReplaceBlocks(ctx, db, contentMatch.ID, []Block{{Type: BlockText, Content: "Usamos kubernetes para desplegar"}}); err != nil {
		t.Fatalf("ReplaceBlocks: %v", err)
	}

	byTitle, err := SearchPages(ctx, db, orgID, "instalación")
	if err != nil || len(byTitle) != 1 || byTitle[0].ID != titleMatch.ID {
		t.Fatalf("expected title match on %s, got %+v, err=%v", titleMatch.ID, byTitle, err)
	}

	byContent, err := SearchPages(ctx, db, orgID, "kubernetes")
	if err != nil || len(byContent) != 1 || byContent[0].ID != contentMatch.ID {
		t.Fatalf("expected content match on %s, got %+v, err=%v", contentMatch.ID, byContent, err)
	}

	none, err := SearchPages(ctx, db, orgID, "no-existe-esto")
	if err != nil || len(none) != 0 {
		t.Fatalf("expected no matches, got %+v, err=%v", none, err)
	}
}

func TestEffectivePageAccessScenarios(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	const orgID = "org-1"

	space, _ := CreateSpace(ctx, db, orgID, "Docs")
	page, err := CreatePage(ctx, db, orgID, space.ID, CategoryReference, "Doc", "user-1")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}

	if access := PageAccess(ctx, db, "admin-user", true, *page); access == nil || *access != groups.AccessEditor {
		t.Fatalf("expected Platform Admin to always have editor access, got %v", access)
	}

	if access := PageAccess(ctx, db, "stranger", false, *page); access != nil {
		t.Fatalf("expected no access for a user with no group grant, got %v", *access)
	}

	spaceGroup, err := groups.CreateGroup(ctx, db, orgID, "Lectores del espacio")
	if err != nil {
		t.Fatalf("CreateGroup spaceGroup: %v", err)
	}
	if err := groups.UpsertMember(ctx, db, spaceGroup.ID, "user-2", groups.AccessReader); err != nil {
		t.Fatalf("UpsertMember user-2: %v", err)
	}
	if err := groups.GrantSpaceAccess(ctx, db, spaceGroup.ID, space.ID); err != nil {
		t.Fatalf("GrantSpaceAccess: %v", err)
	}
	if access := PageAccess(ctx, db, "user-2", false, *page); access == nil || *access != groups.AccessReader {
		t.Fatalf("expected reader access inherited from space grant, got %v", access)
	}

	pageGroup, err := groups.CreateGroup(ctx, db, orgID, "Editor de una página suelta")
	if err != nil {
		t.Fatalf("CreateGroup pageGroup: %v", err)
	}
	if err := groups.UpsertMember(ctx, db, pageGroup.ID, "user-3", groups.AccessEditor); err != nil {
		t.Fatalf("UpsertMember user-3: %v", err)
	}
	if err := groups.GrantPageAccess(ctx, db, pageGroup.ID, page.ID); err != nil {
		t.Fatalf("GrantPageAccess: %v", err)
	}
	if access := PageAccess(ctx, db, "user-3", false, *page); access == nil || *access != groups.AccessEditor {
		t.Fatalf("expected editor access from loose page grant (no space access), got %v", access)
	}
}

func insertOrg(t *testing.T, db *mongo.Database, orgID string) {
	t.Helper()
	_, err := db.Collection("organizations").UpdateOne(context.Background(),
		bson.M{"_id": orgID},
		bson.M{"$setOnInsert": bson.M{"_id": orgID, "storage_used_bytes": int64(0)}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		t.Fatalf("insertOrg: %v", err)
	}
}

func storageUsed(t *testing.T, db *mongo.Database, orgID string) int64 {
	t.Helper()
	var doc struct {
		StorageUsedBytes int64 `bson:"storage_used_bytes"`
	}
	if err := db.Collection("organizations").FindOne(context.Background(), bson.M{"_id": orgID}).Decode(&doc); err != nil {
		t.Fatalf("storageUsed lookup: %v", err)
	}
	return doc.StorageUsedBytes
}
