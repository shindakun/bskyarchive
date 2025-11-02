package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitDB initializes the SQLite database with production settings
func InitDB(path string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connection string with production-ready settings
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _synchronous=NORMAL: Balance between safety and performance (safe for WAL mode)
	// - _busy_timeout=5000: Wait up to 5 seconds if database is locked
	// - _cache=private: Use private page cache (better for single-user app)
	// - _temp_store=memory: Store temporary tables in memory for speed
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache=private&_temp_store=memory", path)

	// Open database
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys (required for referential integrity)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Optional: Set cache size and memory-mapped I/O for performance
	// 64MB cache (-64000 pages of 1KB each)
	if _, err := db.Exec("PRAGMA cache_size = -64000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set cache size: %w", err)
	}

	// 256MB mmap (memory-mapped I/O for faster reads)
	if _, err := db.Exec("PRAGMA mmap_size = 268435456"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set mmap size: %w", err)
	}

	// Set connection pool limits (SQLite needs single writer)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// runMigrations creates all necessary tables and indices
func runMigrations(db *sql.DB) error {
	migrations := []string{
		// Create schema_version table for tracking migrations
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			did TEXT NOT NULL UNIQUE,
			handle TEXT NOT NULL,
			display_name TEXT,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create posts table
		`CREATE TABLE IF NOT EXISTS posts (
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
		)`,

		// Create FTS5 virtual table for full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(
			uri UNINDEXED,
			text,
			content='posts',
			content_rowid='rowid'
		)`,

		// Create triggers to keep FTS index in sync
		`CREATE TRIGGER IF NOT EXISTS posts_ai AFTER INSERT ON posts BEGIN
			INSERT INTO posts_fts(rowid, uri, text)
			VALUES (new.rowid, new.uri, new.text);
		END`,

		`CREATE TRIGGER IF NOT EXISTS posts_ad AFTER DELETE ON posts BEGIN
			DELETE FROM posts_fts WHERE rowid = old.rowid;
		END`,

		`CREATE TRIGGER IF NOT EXISTS posts_au AFTER UPDATE ON posts BEGIN
			UPDATE posts_fts SET text = new.text WHERE rowid = old.rowid;
		END`,

		// Create profiles table
		`CREATE TABLE IF NOT EXISTS profiles (
			did TEXT NOT NULL,
			handle TEXT NOT NULL,
			display_name TEXT,
			description TEXT,
			avatar_url TEXT,
			banner_url TEXT,
			followers_count INTEGER DEFAULT 0,
			follows_count INTEGER DEFAULT 0,
			posts_count INTEGER DEFAULT 0,
			snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (did, snapshot_at)
		)`,

		// Create media table
		`CREATE TABLE IF NOT EXISTS media (
			hash TEXT PRIMARY KEY,
			post_uri TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			file_path TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			width INTEGER,
			height INTEGER,
			alt_text TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_uri) REFERENCES posts(uri) ON DELETE CASCADE
		)`,

		// Create operations table for background tasks
		`CREATE TABLE IF NOT EXISTS operations (
			id TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			progress INTEGER DEFAULT 0,
			total INTEGER DEFAULT 0,
			error TEXT,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			FOREIGN KEY (did) REFERENCES sessions(did) ON DELETE CASCADE
		)`,

		// Create indices for common queries
		`CREATE INDEX IF NOT EXISTS idx_posts_did ON posts(did)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_has_media ON posts(has_media) WHERE has_media = 1`,
		`CREATE INDEX IF NOT EXISTS idx_posts_is_reply ON posts(is_reply) WHERE is_reply = 1`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_did ON profiles(did)`,
		`CREATE INDEX IF NOT EXISTS idx_profiles_snapshot_at ON profiles(snapshot_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_media_post_uri ON media(post_uri)`,
		`CREATE INDEX IF NOT EXISTS idx_operations_did_status ON operations(did, status)`,
	}

	// Execute migrations in a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, migration := range migrations {
		if _, err := tx.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	// Record schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (1)"); err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	// Run incremental migrations
	if err := runIncrementalMigrations(db); err != nil {
		return fmt.Errorf("failed to run incremental migrations: %w", err)
	}

	// Invalidate all existing sessions on startup
	// This ensures clean state after app restarts and prevents issues with stale sessions
	if err := invalidateAllSessions(db); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	return nil
}

// runIncrementalMigrations runs schema updates for existing databases
func runIncrementalMigrations(db *sql.DB) error {
	// Get current schema version
	var currentVersion int
	err := db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Migration 2: Add quote_count column to posts table
	if currentVersion < 2 {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration 2: %w", err)
		}
		defer tx.Rollback()

		// Check if column already exists (in case of partial migration)
		var columnExists bool
		err = tx.QueryRow(`
			SELECT COUNT(*) > 0
			FROM pragma_table_info('posts')
			WHERE name = 'quote_count'
		`).Scan(&columnExists)
		if err != nil {
			return fmt.Errorf("failed to check if quote_count exists: %w", err)
		}

		if !columnExists {
			if _, err := tx.Exec("ALTER TABLE posts ADD COLUMN quote_count INTEGER DEFAULT 0"); err != nil {
				return fmt.Errorf("failed to add quote_count column: %w", err)
			}
		}

		// Update schema version
		if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (2)"); err != nil {
			return fmt.Errorf("failed to update schema version to 2: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration 2: %w", err)
		}
	}

	// Migration 3: Add exports table for export download management
	if currentVersion < 3 {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration 3: %w", err)
		}
		defer tx.Rollback()

		// Check if table already exists (in case of partial migration)
		var tableExists bool
		err = tx.QueryRow(`
			SELECT COUNT(*) > 0
			FROM sqlite_master
			WHERE type = 'table' AND name = 'exports'
		`).Scan(&tableExists)
		if err != nil {
			return fmt.Errorf("failed to check if exports table exists: %w", err)
		}

		if !tableExists {
			// Create exports table
			if _, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS exports (
					id TEXT PRIMARY KEY,
					did TEXT NOT NULL,
					format TEXT NOT NULL,
					created_at INTEGER NOT NULL,
					directory_path TEXT NOT NULL,
					post_count INTEGER NOT NULL,
					media_count INTEGER DEFAULT 0,
					size_bytes INTEGER DEFAULT 0,
					date_range_start INTEGER,
					date_range_end INTEGER,
					manifest_path TEXT,
					CHECK (post_count >= 0),
					CHECK (media_count >= 0),
					CHECK (size_bytes >= 0)
				)
			`); err != nil {
				return fmt.Errorf("failed to create exports table: %w", err)
			}

			// Create indices for query performance
			if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_exports_did ON exports(did)"); err != nil {
				return fmt.Errorf("failed to create idx_exports_did: %w", err)
			}

			if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_exports_created_at ON exports(created_at DESC)"); err != nil {
				return fmt.Errorf("failed to create idx_exports_created_at: %w", err)
			}

			if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_exports_did_created ON exports(did, created_at DESC)"); err != nil {
				return fmt.Errorf("failed to create idx_exports_did_created: %w", err)
			}
		}

		// Update schema version
		if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (3)"); err != nil {
			return fmt.Errorf("failed to update schema version to 3: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration 3: %w", err)
		}
	}

	return nil
}

// invalidateAllSessions clears all sessions on startup
// This ensures a clean state and prevents issues with stale sessions after restarts
func invalidateAllSessions(db *sql.DB) error {
	result, err := db.Exec("DELETE FROM sessions")
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows > 0 {
		fmt.Printf("Invalidated %d existing session(s) on startup\n", rows)
	}

	return nil
}
