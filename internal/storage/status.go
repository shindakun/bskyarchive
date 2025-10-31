package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// parseTimestamp parses SQLite timestamp strings
func parseTimestamp(s string) (time.Time, error) {
	// Try RFC3339 first (standard ISO 8601)
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	// Try SQLite default format
	t, err = time.Parse("2006-01-02 15:04:05", s)
	if err == nil {
		return t, nil
	}

	// Try with timezone
	t, err = time.Parse("2006-01-02 15:04:05-07:00", s)
	if err == nil {
		return t, nil
	}

	// Try Go's time.Time.String() format with timezone and monotonic clock
	// Example: "2025-10-30 18:00:45.077092 -0700 PDT m=+23.090008635"
	// We'll parse just the date/time part before the monotonic clock
	if idx := strings.Index(s, " m="); idx != -1 {
		s = s[:idx] // Remove monotonic clock part
	}
	// Now try parsing: "2025-10-30 18:00:45.077092 -0700 PDT"
	t, err = time.Parse("2006-01-02 15:04:05.999999 -0700 MST", s)
	if err == nil {
		return t, nil
	}

	// Try without fractional seconds
	t, err = time.Parse("2006-01-02 15:04:05 -0700 MST", s)
	return t, err
}

// GetArchiveStatus retrieves aggregated archive status for a user
func GetArchiveStatus(db *sql.DB, did string) (*models.ArchiveStatus, error) {
	status := &models.ArchiveStatus{
		DID: did,
	}

	// Get total posts count
	err := db.QueryRow("SELECT COUNT(*) FROM posts WHERE did = ?", did).Scan(&status.TotalPosts)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	// Get total media count
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT m.hash)
		FROM media m
		JOIN posts p ON m.post_uri = p.uri
		WHERE p.did = ?
	`, did).Scan(&status.TotalMedia)
	if err != nil {
		return nil, fmt.Errorf("failed to count media: %w", err)
	}

	// Get oldest and newest posts
	var oldestPostStr, newestPostStr sql.NullString
	err = db.QueryRow(`
		SELECT MIN(created_at), MAX(created_at)
		FROM posts
		WHERE did = ?
	`, did).Scan(&oldestPostStr, &newestPostStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get post date range: %w", err)
	}
	if oldestPostStr.Valid && oldestPostStr.String != "" {
		if t, err := parseTimestamp(oldestPostStr.String); err == nil {
			status.OldestPost = &t
		}
	}
	if newestPostStr.Valid && newestPostStr.String != "" {
		if t, err := parseTimestamp(newestPostStr.String); err == nil {
			status.NewestPost = &t
		}
	}

	// Get last archive timestamp (most recent operation start time)
	var lastArchiveStr sql.NullString
	err = db.QueryRow(`
		SELECT MAX(started_at)
		FROM operations
		WHERE did = ?
	`, did).Scan(&lastArchiveStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get last archive time: %w", err)
	}
	if lastArchiveStr.Valid && lastArchiveStr.String != "" {
		if t, err := parseTimestamp(lastArchiveStr.String); err == nil {
			status.LastArchiveAt = &t
		}
	}

	// Get last successful archive
	var lastSuccessfulStr sql.NullString
	err = db.QueryRow(`
		SELECT MAX(completed_at)
		FROM operations
		WHERE did = ? AND status = 'completed'
	`, did).Scan(&lastSuccessfulStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get last successful archive: %w", err)
	}
	if lastSuccessfulStr.Valid && lastSuccessfulStr.String != "" {
		if t, err := parseTimestamp(lastSuccessfulStr.String); err == nil {
			status.LastSuccessfulAt = &t
		}
	}

	// Get total archive size
	err = db.QueryRow(`
		SELECT COALESCE(SUM(DISTINCT m.size_bytes), 0)
		FROM media m
		JOIN posts p ON m.post_uri = p.uri
		WHERE p.did = ?
	`, did).Scan(&status.TotalArchiveSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate archive size: %w", err)
	}

	// Get posts with media count
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE did = ? AND has_media = 1
	`, did).Scan(&status.PostsWithMedia)
	if err != nil {
		return nil, fmt.Errorf("failed to count posts with media: %w", err)
	}

	// Get replies count
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM posts
		WHERE did = ? AND is_reply = 1
	`, did).Scan(&status.RepliesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count replies: %w", err)
	}

	// Get media breakdown
	status.MediaBreakdown = &models.MediaBreakdown{}
	err = db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN m.mime_type LIKE 'image/%' THEN 1 ELSE 0 END), 0) as images,
			COALESCE(SUM(CASE WHEN m.mime_type LIKE 'video/%' THEN 1 ELSE 0 END), 0) as videos,
			COALESCE(SUM(CASE WHEN m.mime_type NOT LIKE 'image/%' AND m.mime_type NOT LIKE 'video/%' THEN 1 ELSE 0 END), 0) as other
		FROM media m
		JOIN posts p ON m.post_uri = p.uri
		WHERE p.did = ?
	`, did).Scan(&status.MediaBreakdown.Images, &status.MediaBreakdown.Videos, &status.MediaBreakdown.Other)
	if err != nil {
		return nil, fmt.Errorf("failed to get media breakdown: %w", err)
	}

	// Get engagement summary
	status.EngagementSummary = &models.EngagementSummary{}
	err = db.QueryRow(`
		SELECT
			COALESCE(SUM(like_count), 0) as total_likes,
			COALESCE(SUM(repost_count), 0) as total_reposts,
			COALESCE(SUM(reply_count), 0) as total_replies,
			COALESCE(AVG(like_count), 0) as avg_likes,
			COALESCE(AVG(repost_count), 0) as avg_reposts,
			COALESCE(AVG(reply_count), 0) as avg_replies
		FROM posts
		WHERE did = ?
	`, did).Scan(
		&status.EngagementSummary.TotalLikes,
		&status.EngagementSummary.TotalReposts,
		&status.EngagementSummary.TotalReplies,
		&status.EngagementSummary.AvgLikes,
		&status.EngagementSummary.AvgReposts,
		&status.EngagementSummary.AvgReplies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement summary: %w", err)
	}

	// Get active operation
	activeOp, err := GetActiveOperation(db, did)
	if err != nil {
		return nil, fmt.Errorf("failed to get active operation: %w", err)
	}
	status.ActiveOperation = activeOp

	// Get recent operations
	recentOps, err := ListRecentOperations(db, did, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent operations: %w", err)
	}
	status.RecentOperations = recentOps

	return status, nil
}

// GetArchiveStatusSimple retrieves a simplified archive status (faster query)
func GetArchiveStatusSimple(db *sql.DB, did string) (*models.ArchiveStatus, error) {
	status := &models.ArchiveStatus{
		DID: did,
	}

	// Single query to get basic stats
	var oldestPostStr, newestPostStr sql.NullString
	err := db.QueryRow(`
		SELECT
			COUNT(*) as total_posts,
			MIN(created_at) as oldest_post,
			MAX(created_at) as newest_post,
			COALESCE(SUM(CASE WHEN has_media = 1 THEN 1 ELSE 0 END), 0) as posts_with_media,
			COALESCE(SUM(CASE WHEN is_reply = 1 THEN 1 ELSE 0 END), 0) as replies
		FROM posts
		WHERE did = ?
	`, did).Scan(
		&status.TotalPosts,
		&oldestPostStr,
		&newestPostStr,
		&status.PostsWithMedia,
		&status.RepliesCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic stats: %w", err)
	}

	// Parse timestamp strings
	if oldestPostStr.Valid && oldestPostStr.String != "" {
		if t, err := parseTimestamp(oldestPostStr.String); err == nil {
			status.OldestPost = &t
		}
	}
	if newestPostStr.Valid && newestPostStr.String != "" {
		if t, err := parseTimestamp(newestPostStr.String); err == nil {
			status.NewestPost = &t
		}
	}

	// Get active operation
	activeOp, err := GetActiveOperation(db, did)
	if err != nil {
		return nil, fmt.Errorf("failed to get active operation: %w", err)
	}
	status.ActiveOperation = activeOp

	return status, nil
}
