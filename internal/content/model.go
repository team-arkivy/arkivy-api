// Package content implements the "Content" module described in
// docs/ARCHITECTURE.md: Spaces, Pages, typed Blocks and Attachments
// (docs/DATABASE.md §3 and §5), adapted to Mongo. It's the only module with
// write access to real documentation — every read/write goes through the
// permission check in access.go, which reuses internal/groups' effective-
// access calculation from Fase 2.
package content

import "time"

type Category string

const (
	CategoryTutorial    Category = "tutorial"
	CategoryHowTo       Category = "how_to"
	CategoryReference   Category = "reference"
	CategoryExplanation Category = "explanation"
	CategoryMultiple    Category = "multiple"
)

type BlockType string

const (
	BlockText BlockType = "text"
	BlockCode BlockType = "code"
	BlockFile BlockType = "file"
	BlockLink BlockType = "link"
)

type FileType string

const (
	FilePDF      FileType = "pdf"
	FileWord     FileType = "word"
	FileTxt      FileType = "txt"
	FileMarkdown FileType = "markdown"
	FileExcel    FileType = "excel"
)

// MaxAttachmentBytes is the fixed 25MB per-file cap from RF-DOC-11,
// independent of the organization's plan.
const MaxAttachmentBytes int64 = 25 * 1024 * 1024

// Space mirrors DATABASE.md's `spaces` table.
type Space struct {
	ID             string    `bson:"_id"             json:"id"`
	OrganizationID string    `bson:"organization_id" json:"organizationId"`
	Name           string    `bson:"name"             json:"name"`
	CreatedAt      time.Time `bson:"created_at"       json:"createdAt"`
}

// Page mirrors DATABASE.md's `pages` table. OrganizationID is duplicated
// (like DATABASE.md already does on attachments) so permission checks and
// storage accounting don't need a space → org lookup on every request.
type Page struct {
	ID             string    `bson:"_id"             json:"id"`
	SpaceID        string    `bson:"space_id"        json:"spaceId"`
	OrganizationID string    `bson:"organization_id" json:"organizationId"`
	Category       Category  `bson:"category"        json:"category"`
	OrderIndex     int       `bson:"order_index"     json:"orderIndex"`
	Title          string    `bson:"title"           json:"title"`
	Status         string    `bson:"status"          json:"status"` // fixed "draft" until Fase 4
	SourceType     string    `bson:"source_type"     json:"sourceType"` // fixed "manual" until RF-ING
	AuthorID       string    `bson:"author_id"       json:"authorId"`
	CreatedAt      time.Time `bson:"created_at"       json:"createdAt"`
	UpdatedAt      time.Time `bson:"updated_at"       json:"updatedAt"`
}

// Block mirrors DATABASE.md's `blocks` table.
type Block struct {
	ID         string         `bson:"id"          json:"id"`
	OrderIndex int            `bson:"order_index" json:"orderIndex"`
	Type       BlockType      `bson:"type"         json:"type"`
	Content    string         `bson:"content"      json:"content"`
	Language   *string        `bson:"language,omitempty" json:"language,omitempty"`
	Metadata   map[string]any `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// pageBlocks is the Mongo document shape for the blocks collection: one
// document per page holding its full ordered block array (see ReplaceBlocks
// in store.go for why blocks are replaced as a unit instead of row by row).
type pageBlocks struct {
	PageID string  `bson:"_id"`
	Blocks []Block `bson:"blocks"`
}

// Attachment mirrors DATABASE.md's `attachments` table. StorageRef is the
// GridFS file ID (hex-encoded ObjectID) instead of DATABASE.md's storage_url
// — see the Fase 3 plan for why GridFS was chosen over S3 for the MVP.
type Attachment struct {
	ID             string    `bson:"_id"             json:"id"`
	BlockID        string    `bson:"block_id"        json:"blockId"`
	OrganizationID string    `bson:"organization_id" json:"organizationId"`
	FileName       string    `bson:"file_name"       json:"fileName"`
	FileType       FileType  `bson:"file_type"       json:"fileType"`
	FileSizeBytes  int64     `bson:"file_size_bytes" json:"fileSizeBytes"`
	StorageRef     string    `bson:"storage_ref"     json:"storageRef"`
	UploadedBy     string    `bson:"uploaded_by"     json:"uploadedBy"`
	CreatedAt      time.Time `bson:"created_at"       json:"createdAt"`
}

// TocCategory groups a Space's pages by category for the table of contents
// (RF-DOC-02).
type TocCategory struct {
	Category Category `json:"category"`
	Pages    []Page   `json:"pages"`
}
