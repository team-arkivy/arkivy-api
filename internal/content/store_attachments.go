package content

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"arkivy-api/internal/organizations"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// attachmentTypeByExt maps the extensions allowed by RF-DOC-11 to their
// FileType. Anything else is rejected.
var attachmentTypeByExt = map[string]FileType{
	".pdf":  FilePDF,
	".doc":  FileWord,
	".docx": FileWord,
	".txt":  FileTxt,
	".md":   FileMarkdown,
	".xls":  FileExcel,
	".xlsx": FileExcel,
}

var ErrUnsupportedFileType = fmt.Errorf("unsupported file type — allowed: pdf, doc/docx, txt, md, xls/xlsx")
var ErrFileTooLarge = fmt.Errorf("file exceeds the 25MB limit")

// UploadAttachment validates fileName's extension and sizeBytes against
// RF-DOC-11, stores the file in GridFS, records the Attachment, and credits
// orgID's storage_used_bytes counter (organizations.IncrementStorageUsed)
// ahead of Fase 6's plan-limit enforcement.
func UploadAttachment(ctx context.Context, db *mongo.Database, bucket *mongo.GridFSBucket, orgID, blockID, fileName string, sizeBytes int64, uploaderID string, reader io.Reader) (*Attachment, error) {
	if sizeBytes > MaxAttachmentBytes {
		return nil, ErrFileTooLarge
	}

	fileType, ok := attachmentTypeByExt[strings.ToLower(filepath.Ext(fileName))]
	if !ok {
		return nil, ErrUnsupportedFileType
	}

	fileID, err := bucket.UploadFromStream(ctx, fileName, reader)
	if err != nil {
		return nil, err
	}

	a := Attachment{
		ID:             uuid.New().String(),
		BlockID:        blockID,
		OrganizationID: orgID,
		FileName:       fileName,
		FileType:       fileType,
		FileSizeBytes:  sizeBytes,
		StorageRef:     fileID.Hex(),
		UploadedBy:     uploaderID,
		CreatedAt:      time.Now(),
	}
	if _, err := db.Collection(attachmentsCollection).InsertOne(ctx, a); err != nil {
		return nil, err
	}

	if err := organizations.IncrementStorageUsed(ctx, db, orgID, sizeBytes); err != nil {
		return nil, err
	}

	return &a, nil
}

// GetAttachment returns an Attachment's record by ID.
func GetAttachment(ctx context.Context, db *mongo.Database, attachmentID string) (*Attachment, error) {
	var a Attachment
	err := db.Collection(attachmentsCollection).FindOne(ctx, bson.M{"_id": attachmentID}).Decode(&a)
	if err == mongo.ErrNoDocuments {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAttachmentsByBlock returns every Attachment on blockID.
func ListAttachmentsByBlock(ctx context.Context, db *mongo.Database, blockID string) ([]Attachment, error) {
	cur, err := db.Collection(attachmentsCollection).Find(ctx, bson.M{"block_id": blockID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	attachments := []Attachment{}
	if err := cur.All(ctx, &attachments); err != nil {
		return nil, err
	}
	return attachments, nil
}

// OpenAttachmentDownload returns an Attachment's record and an open GridFS
// download stream positioned at its content. The caller is responsible for
// closing the stream.
func OpenAttachmentDownload(ctx context.Context, db *mongo.Database, bucket *mongo.GridFSBucket, attachmentID string) (*Attachment, *mongo.GridFSDownloadStream, error) {
	a, err := GetAttachment(ctx, db, attachmentID)
	if err != nil {
		return nil, nil, err
	}

	fileID, err := bson.ObjectIDFromHex(a.StorageRef)
	if err != nil {
		return nil, nil, err
	}

	stream, err := bucket.OpenDownloadStream(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	return a, stream, nil
}

// DeleteAttachment removes an Attachment's GridFS file and record, and
// debits orgID's storage_used_bytes counter.
func DeleteAttachment(ctx context.Context, db *mongo.Database, bucket *mongo.GridFSBucket, attachmentID string) error {
	a, err := GetAttachment(ctx, db, attachmentID)
	if err != nil {
		return err
	}

	fileID, err := bson.ObjectIDFromHex(a.StorageRef)
	if err != nil {
		return err
	}
	if err := bucket.Delete(ctx, fileID); err != nil {
		return err
	}

	if _, err := db.Collection(attachmentsCollection).DeleteOne(ctx, bson.M{"_id": attachmentID}); err != nil {
		return err
	}

	return organizations.IncrementStorageUsed(ctx, db, a.OrganizationID, -a.FileSizeBytes)
}
