package main

import (
	"time"
)

// === Node Types ===
const (
	NodeTypeNote        = "note"
	NodeTypePage        = "page"
	NodeTypePost        = "post"
	NodeTypeCanvas      = "canvas"
	NodeTypeShaderDemo  = "shader-demo"
	NodeTypeCodeSnippet = "code-snippet"
	NodeTypeImage       = "image"
	NodeTypeVideo       = "video"
	NodeTypeAudio       = "audio"
	NodeTypeDocument    = "document"
	NodeTypeTodo        = "todo"
	NodeTypeReminder    = "reminder"
)

// === Types ===
type Node struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	ParentID     string    `json:"parent_id,omitempty"`
	Path         string    `json:"path"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Slug         string    `json:"slug,omitempty"`
	CanonicalURI string    `json:"canonical_uri,omitempty"`
	Body         string    `json:"body,omitempty"`     // JSON structured body
	Metadata     string    `json:"metadata,omitempty"` // JSON metadata
	MimeType     string    `json:"mime_type"`
	CreatedAt    time.Time `json:"created_at"`
	ModifiedAt   time.Time `json:"modified_at"`
	Tags         []string  `json:"tags,omitempty"`
	References   []string  `json:"references,omitempty"`
	Visibility   string    `json:"visibility,omitempty"`
	Status       string    `json:"status,omitempty"`
	SiteID       string    `json:"site_id,omitempty"`
}

type Version struct {
	ID            string     `json:"id"`
	NodeID        string     `json:"node_id"`
	VersionNumber int        `json:"version_number"`
	Content       string     `json:"content"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ModifiedAt    time.Time  `json:"modified_at"`
	IsCurrent     bool       `json:"is_current"`
}

type BlogPost struct {
	ID          string     `json:"id"`
	NodeID      string     `json:"node_id"`
	Slug        string     `json:"slug"`
	Excerpt     string     `json:"excerpt"`
	PublishDate *time.Time `json:"publish_date,omitempty"`
	Category    string     `json:"category"`
}

type MediaFile struct {
	ID               string    `json:"id"`
	NodeID           string    `json:"node_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	Checksum         string    `json:"checksum"`
	StorageURL       string    `json:"storage_url"`
	UploadedBy       string    `json:"uploaded_by"`
	CreatedAt        time.Time `json:"created_at"`
}

type Reference struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	LinkType     string `json:"link_type"`
	LinkText     string `json:"link_text"`
}

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Citation struct {
	ID             string `json:"id"`
	NodeID         string `json:"node_id"`
	CitationKey    string `json:"citation_key"`
	Authors        string `json:"authors"`
	Title          string `json:"title"`
	Year           int    `json:"year"`
	Publisher      string `json:"publisher"`
	URL            string `json:"url"`
	DOI            string `json:"doi"`
	CitationFormat string `json:"citation_format"`
	RawBibtex      string `json:"raw_bibtex"`
}

type Site struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // project, portfolio, blog, etc
	CreatedAt   time.Time `json:"created_at"`
	ModifiedAt  time.Time `json:"modified_at"`
}

type NodeURI struct {
	ID        string    `json:"id"`
	NodeID    string    `json:"node_id"`
	URI       string    `json:"uri"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

type PluginManifest struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Manifest  string    `json:"manifest"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
