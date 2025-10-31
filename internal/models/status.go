package models

import (
	"time"
)

// ArchiveStatus represents the current state of a user's archive
// This is derived data aggregated from posts, media, and operations tables
type ArchiveStatus struct {
	DID                 string                `json:"did"`
	TotalPosts          int64                 `json:"total_posts"`
	TotalMedia          int64                 `json:"total_media"`
	OldestPost          *time.Time            `json:"oldest_post,omitempty"`
	NewestPost          *time.Time            `json:"newest_post,omitempty"`
	LastArchiveAt       *time.Time            `json:"last_archive_at,omitempty"`
	LastSuccessfulAt    *time.Time            `json:"last_successful_at,omitempty"`
	TotalArchiveSize    int64                 `json:"total_archive_size_bytes"` // Total size in bytes
	ActiveOperation     *ArchiveOperation     `json:"active_operation,omitempty"`
	RecentOperations    []ArchiveOperation    `json:"recent_operations,omitempty"` // Last 5 operations
	PostsWithMedia      int64                 `json:"posts_with_media"`
	RepliesCount        int64                 `json:"replies_count"`
	MediaBreakdown      *MediaBreakdown       `json:"media_breakdown,omitempty"`
	EngagementSummary   *EngagementSummary    `json:"engagement_summary,omitempty"`
}

// MediaBreakdown provides statistics about media types
type MediaBreakdown struct {
	Images int64 `json:"images"`
	Videos int64 `json:"videos"`
	Other  int64 `json:"other"`
}

// EngagementSummary provides aggregate engagement statistics
type EngagementSummary struct {
	TotalLikes   int64 `json:"total_likes"`
	TotalReposts int64 `json:"total_reposts"`
	TotalReplies int64 `json:"total_replies"`
	AvgLikes     float64 `json:"avg_likes"`
	AvgReposts   float64 `json:"avg_reposts"`
	AvgReplies   float64 `json:"avg_replies"`
}

// HasActiveOperation checks if there is currently an active archive operation
func (s *ArchiveStatus) HasActiveOperation() bool {
	return s.ActiveOperation != nil && s.ActiveOperation.IsActive()
}

// IsEmpty checks if the archive has no content
func (s *ArchiveStatus) IsEmpty() bool {
	return s.TotalPosts == 0 && s.TotalMedia == 0
}

// ArchiveSizeGB returns the archive size in gigabytes
func (s *ArchiveStatus) ArchiveSizeGB() float64 {
	return float64(s.TotalArchiveSize) / (1024 * 1024 * 1024)
}

// ArchiveSizeMB returns the archive size in megabytes
func (s *ArchiveStatus) ArchiveSizeMB() float64 {
	return float64(s.TotalArchiveSize) / (1024 * 1024)
}
