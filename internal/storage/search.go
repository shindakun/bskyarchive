package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shindakun/bskyarchive/internal/models"
)

// SearchPosts performs full-text search using FTS5, or direct URI lookup for AT protocol URIs
func SearchPosts(db *sql.DB, did, query string, limit, offset int) (*models.SearchPostsResponse, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	if limit <= 0 || limit > 100 {
		limit = 20 // Default page size
	}
	if offset < 0 {
		offset = 0
	}

	// Check if query is an AT protocol URI
	if strings.HasPrefix(query, "at://") {
		// Direct URI lookup
		post, err := GetPost(db, query)
		if err != nil {
			// Post not found, return empty results
			return &models.SearchPostsResponse{
				Posts:   []models.Post{},
				Query:   query,
				Total:   0,
				Elapsed: "",
			}, nil
		}

		// Return single post result
		return &models.SearchPostsResponse{
			Posts:   []models.Post{*post},
			Query:   query,
			Total:   1,
			Elapsed: "",
		}, nil
	}

	// Get total count of matching posts
	countQuery := `
		SELECT COUNT(*)
		FROM posts_fts
		WHERE posts_fts MATCH ?
		AND uri IN (SELECT uri FROM posts WHERE did = ?)
	`
	var total int64
	err := db.QueryRow(countQuery, query, did).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	// Search posts using FTS5
	searchQuery := `
		SELECT p.uri, p.cid, p.did, p.text, p.created_at, p.indexed_at,
			   p.has_media, p.like_count, p.repost_count, p.reply_count,
			   p.is_reply, p.reply_parent, p.embed_type, p.embed_data, p.labels, p.archived_at
		FROM posts_fts
		JOIN posts p ON posts_fts.uri = p.uri
		WHERE posts_fts MATCH ?
		AND p.did = ?
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(searchQuery, query, did, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var embedData, labels []byte

		err := rows.Scan(
			&post.URI, &post.CID, &post.DID, &post.Text, &post.CreatedAt, &post.IndexedAt,
			&post.HasMedia, &post.LikeCount, &post.RepostCount, &post.ReplyCount,
			&post.IsReply, &post.ReplyParent, &post.EmbedType, &embedData, &labels, &post.ArchivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
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
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return &models.SearchPostsResponse{
		Posts:   posts,
		Query:   query,
		Total:   int(total),
		Elapsed: "", // Set by caller if needed
	}, nil
}
