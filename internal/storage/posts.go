package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shindakun/bskyarchive/internal/models"
)

// SavePost inserts or updates a post in the database
func SavePost(db *sql.DB, post *models.Post) error {
	if err := post.Validate(); err != nil {
		return fmt.Errorf("invalid post: %w", err)
	}

	// Serialize JSON fields
	embedData, err := json.Marshal(post.EmbedData)
	if err != nil {
		return fmt.Errorf("failed to marshal embed_data: %w", err)
	}

	labels, err := json.Marshal(post.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	query := `
		INSERT INTO posts (
			uri, cid, did, text, created_at, indexed_at,
			has_media, like_count, repost_count, reply_count, quote_count,
			is_reply, reply_parent, embed_type, embed_data, labels, archived_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(uri) DO UPDATE SET
			cid = excluded.cid,
			text = excluded.text,
			indexed_at = excluded.indexed_at,
			has_media = excluded.has_media,
			like_count = excluded.like_count,
			repost_count = excluded.repost_count,
			reply_count = excluded.reply_count,
			quote_count = excluded.quote_count,
			embed_type = excluded.embed_type,
			embed_data = excluded.embed_data,
			labels = excluded.labels
	`

	_, err = db.Exec(query,
		post.URI, post.CID, post.DID, post.Text, post.CreatedAt, post.IndexedAt,
		post.HasMedia, post.LikeCount, post.RepostCount, post.ReplyCount, post.QuoteCount,
		post.IsReply, post.ReplyParent, post.EmbedType, embedData, labels, post.ArchivedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save post: %w", err)
	}

	return nil
}

// GetPost retrieves a post by its URI
func GetPost(db *sql.DB, uri string) (*models.Post, error) {
	query := `
		SELECT uri, cid, did, text, created_at, indexed_at,
			   has_media, like_count, repost_count, reply_count, quote_count,
			   is_reply, reply_parent, embed_type, embed_data, labels, archived_at
		FROM posts
		WHERE uri = ?
	`

	var post models.Post
	var embedData, labels []byte

	err := db.QueryRow(query, uri).Scan(
		&post.URI, &post.CID, &post.DID, &post.Text, &post.CreatedAt, &post.IndexedAt,
		&post.HasMedia, &post.LikeCount, &post.RepostCount, &post.ReplyCount, &post.QuoteCount,
		&post.IsReply, &post.ReplyParent, &post.EmbedType, &embedData, &labels, &post.ArchivedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("post not found: %s", uri)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	// Deserialize JSON fields
	if len(embedData) > 0 {
		post.EmbedData = json.RawMessage(embedData)
	}
	if len(labels) > 0 {
		post.Labels = json.RawMessage(labels)
	}

	return &post, nil
}

// ListPosts retrieves posts with pagination
func ListPosts(db *sql.DB, did string, limit, offset int) (*models.PagedPostsResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20 // Default page size
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count - if did is empty, count all posts
	var total int64
	var countQuery string
	var err error
	if did == "" {
		countQuery = `SELECT COUNT(*) FROM posts`
		err = db.QueryRow(countQuery).Scan(&total)
	} else {
		countQuery = `SELECT COUNT(*) FROM posts WHERE did = ?`
		err = db.QueryRow(countQuery, did).Scan(&total)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to count posts: %w", err)
	}

	// Get posts - if did is empty, get all posts
	var query string
	var rows *sql.Rows
	if did == "" {
		query = `
			SELECT uri, cid, did, text, created_at, indexed_at,
				   has_media, like_count, repost_count, reply_count, quote_count,
				   is_reply, reply_parent, embed_type, embed_data, labels, archived_at
			FROM posts
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = db.Query(query, limit, offset)
	} else {
		query = `
			SELECT uri, cid, did, text, created_at, indexed_at,
				   has_media, like_count, repost_count, reply_count, quote_count,
				   is_reply, reply_parent, embed_type, embed_data, labels, archived_at
			FROM posts
			WHERE did = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		rows, err = db.Query(query, did, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var embedData, labels []byte

		err := rows.Scan(
			&post.URI, &post.CID, &post.DID, &post.Text, &post.CreatedAt, &post.IndexedAt,
			&post.HasMedia, &post.LikeCount, &post.RepostCount, &post.ReplyCount, &post.QuoteCount,
			&post.IsReply, &post.ReplyParent, &post.EmbedType, &embedData, &labels, &post.ArchivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Deserialize JSON fields
		if len(embedData) > 0 {
			post.EmbedData = json.RawMessage(embedData)
		}
		if len(labels) > 0 {
			post.Labels = json.RawMessage(labels)
		}

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	// Calculate pagination metadata
	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &models.PagedPostsResponse{
		Posts:      posts,
		Total:      int(total),
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}, nil
}

// ListPostsWithDateRange retrieves posts with optional date range filtering
// If dateRange is nil, behaves like ListPosts
func ListPostsWithDateRange(db *sql.DB, did string, dateRange *models.DateRange, limit, offset int) ([]models.Post, error) {
	if limit <= 0 {
		limit = 1000 // Default for exports (larger than browse pagination)
	}
	if offset < 0 {
		offset = 0
	}

	// Build query with date range filters
	var query string
	var args []interface{}

	selectClause := `
		SELECT uri, cid, did, text, created_at, indexed_at,
			   has_media, like_count, repost_count, reply_count, quote_count,
			   is_reply, reply_parent, embed_type, embed_data, labels, archived_at
		FROM posts
	`

	whereConditions := []string{}

	if did != "" {
		whereConditions = append(whereConditions, "did = ?")
		args = append(args, did)
	}

	if dateRange != nil {
		if !dateRange.StartDate.IsZero() {
			whereConditions = append(whereConditions, "created_at >= ?")
			args = append(args, dateRange.StartDate)
		}
		if !dateRange.EndDate.IsZero() {
			whereConditions = append(whereConditions, "created_at <= ?")
			args = append(args, dateRange.EndDate)
		}
	}

	if len(whereConditions) > 0 {
		query = selectClause + " WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			query += " AND " + whereConditions[i]
		}
	} else {
		query = selectClause
	}

	// Add deterministic ordering for stable pagination
	// uri is the primary key, so it acts as a tie-breaker when created_at values are identical
	query += " ORDER BY created_at DESC, uri ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts with date range: %w", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var embedData, labels []byte

		err := rows.Scan(
			&post.URI, &post.CID, &post.DID, &post.Text, &post.CreatedAt, &post.IndexedAt,
			&post.HasMedia, &post.LikeCount, &post.RepostCount, &post.ReplyCount, &post.QuoteCount,
			&post.IsReply, &post.ReplyParent, &post.EmbedType, &embedData, &labels, &post.ArchivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		// Deserialize JSON fields
		if len(embedData) > 0 {
			post.EmbedData = json.RawMessage(embedData)
		}
		if len(labels) > 0 {
			post.Labels = json.RawMessage(labels)
		}

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	return posts, nil
}
