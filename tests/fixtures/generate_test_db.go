package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	var (
		postCount  = flag.Int("posts", 10000, "Number of test posts to generate")
		outputPath = flag.String("output", "test_archive.db", "Output database file path")
		did        = flag.String("did", "did:plc:test123456789", "DID to use for test posts")
	)
	flag.Parse()

	// Remove existing database if it exists
	if err := os.Remove(*outputPath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to remove existing database: %v", err)
	}

	// Create new database
	db, err := sql.Open("sqlite", *outputPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Generate test posts
	log.Printf("Generating %d test posts for DID %s...", *postCount, *did)
	if err := generatePosts(db, *did, *postCount); err != nil {
		log.Fatalf("Failed to generate posts: %v", err)
	}

	log.Printf("✓ Successfully generated %d posts in %s", *postCount, *outputPath)
}

func initSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS posts (
			uri TEXT PRIMARY KEY,
			cid TEXT NOT NULL,
			did TEXT NOT NULL,
			text TEXT,
			created_at TIMESTAMP NOT NULL,
			indexed_at TIMESTAMP NOT NULL,
			has_media BOOLEAN DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			repost_count INTEGER DEFAULT 0,
			reply_count INTEGER DEFAULT 0,
			quote_count INTEGER DEFAULT 0,
			is_reply BOOLEAN DEFAULT 0,
			reply_parent TEXT,
			embed_type TEXT,
			embed_data JSON,
			labels JSON,
			archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_posts_did ON posts(did);
		CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
	`

	_, err := db.Exec(schema)
	return err
}

func generatePosts(db *sql.DB, did string, count int) error {
	// Use transaction for better performance
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at,
		                   has_media, like_count, repost_count, reply_count,
		                   quote_count, is_reply, reply_parent, embed_type, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	// Generate posts with timestamps spread over past year
	now := time.Now()
	baseTime := now.AddDate(0, -12, 0) // 1 year ago

	for i := 0; i < count; i++ {
		// Generate unique identifiers
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%d", did, i)
		cid := fmt.Sprintf("bafyreiabc%015d", i)

		// Calculate timestamp (evenly distributed over past year)
		offsetMinutes := int64(i) * (365 * 24 * 60 / int64(count))
		createdAt := baseTime.Add(time.Duration(offsetMinutes) * time.Minute)
		indexedAt := createdAt.Add(5 * time.Minute) // Indexed 5 minutes after creation

		// Generate text with variation
		var text string
		switch i % 5 {
		case 0:
			text = fmt.Sprintf("Test post #%d: This is a short post.", i)
		case 1:
			text = fmt.Sprintf("Test post #%d: This is a medium length post with some additional content to test text handling and search functionality.", i)
		case 2:
			text = fmt.Sprintf("Test post #%d: Long post with extended content. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam.", i)
		case 3:
			text = fmt.Sprintf("Test post #%d: Post with special characters! @mentions #hashtags https://example.com", i)
		case 4:
			text = fmt.Sprintf("Test post #%d: 🎉 Unicode emoji support test 🚀 ✨ 🎨", i)
		}

		// Vary engagement metrics
		hasMedia := i%10 == 0
		likeCount := i % 100
		repostCount := i % 50
		replyCount := i % 30
		quoteCount := i % 20
		isReply := i%15 == 0

		_, err := stmt.Exec(
			uri, cid, did, text, createdAt, indexedAt,
			hasMedia, likeCount, repostCount, replyCount,
			quoteCount, isReply, "", "", now,
		)
		if err != nil {
			return fmt.Errorf("insert post %d: %w", i, err)
		}

		// Progress logging
		if (i+1)%1000 == 0 {
			log.Printf("Generated %d/%d posts...", i+1, count)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
