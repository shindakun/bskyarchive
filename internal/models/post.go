package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Post represents a single Bluesky post with all metadata, engagement metrics, and relationships
type Post struct {
	URI         string          `json:"uri" db:"uri"`
	CID         string          `json:"cid" db:"cid"`
	DID         string          `json:"did" db:"did"`
	Text        string          `json:"text" db:"text"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	IndexedAt   time.Time       `json:"indexed_at" db:"indexed_at"`
	HasMedia    bool            `json:"has_media" db:"has_media"`
	LikeCount   int             `json:"like_count" db:"like_count"`
	RepostCount int             `json:"repost_count" db:"repost_count"`
	ReplyCount  int             `json:"reply_count" db:"reply_count"`
	QuoteCount  int             `json:"quote_count" db:"quote_count"`
	IsReply     bool            `json:"is_reply" db:"is_reply"`
	ReplyParent string          `json:"reply_parent,omitempty" db:"reply_parent"`
	EmbedType   string          `json:"embed_type,omitempty" db:"embed_type"`
	EmbedData   json.RawMessage `json:"embed_data,omitempty" db:"embed_data"`
	Labels      json.RawMessage `json:"labels,omitempty" db:"labels"`
	ArchivedAt  time.Time       `json:"archived_at" db:"archived_at"`
}

// Validate checks if the post fields are valid
func (p *Post) Validate() error {
	if p.URI == "" {
		return fmt.Errorf("uri is required")
	}

	if !strings.HasPrefix(p.URI, "at://") {
		return fmt.Errorf("uri must start with 'at://'")
	}

	if p.CID == "" {
		return fmt.Errorf("cid is required")
	}

	if p.DID == "" {
		return fmt.Errorf("did is required")
	}

	// Bluesky posts can be up to 300 graphemes, but we're archiving existing posts
	// so we shouldn't validate the length - just store what we receive
	// The actual limit is enforced by Bluesky when creating posts, not when archiving

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}

	if p.IndexedAt.IsZero() {
		return fmt.Errorf("indexed_at is required")
	}

	validEmbedTypes := []string{"images", "external", "record", "record_with_media", ""}
	if p.EmbedType != "" {
		found := false
		for _, et := range validEmbedTypes {
			if p.EmbedType == et {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("embed_type must be one of: images, external, record, record_with_media")
		}
	}

	return nil
}

// PagedPostsResponse represents a paginated list of posts
type PagedPostsResponse struct {
	Posts      []Post `json:"posts"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// SearchPostsResponse represents search results with highlighting
type SearchPostsResponse struct {
	Posts   []Post `json:"posts"`
	Total   int    `json:"total"`
	Query   string `json:"query"`
	Elapsed string `json:"elapsed"` // Search duration
}
