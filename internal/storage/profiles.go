package storage

import (
	"database/sql"
	"fmt"

	"github.com/shindakun/bskyarchive/internal/models"
)

// SaveProfile saves a profile snapshot to the database
func SaveProfile(db *sql.DB, profile *models.Profile) error {
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	query := `
		INSERT INTO profiles (
			did, handle, display_name, description, avatar_url, banner_url,
			followers_count, follows_count, posts_count, snapshot_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		profile.DID, profile.Handle, profile.DisplayName, profile.Description,
		profile.AvatarURL, profile.BannerURL, profile.FollowersCount,
		profile.FollowsCount, profile.PostsCount, profile.SnapshotAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	return nil
}

// GetLatestProfile retrieves the most recent profile snapshot for a DID
func GetLatestProfile(db *sql.DB, did string) (*models.Profile, error) {
	query := `
		SELECT did, handle, display_name, description, avatar_url, banner_url,
			   followers_count, follows_count, posts_count, snapshot_at
		FROM profiles
		WHERE did = ?
		ORDER BY snapshot_at DESC
		LIMIT 1
	`

	var profile models.Profile
	err := db.QueryRow(query, did).Scan(
		&profile.DID, &profile.Handle, &profile.DisplayName, &profile.Description,
		&profile.AvatarURL, &profile.BannerURL, &profile.FollowersCount,
		&profile.FollowsCount, &profile.PostsCount, &profile.SnapshotAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found for DID: %s", did)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &profile, nil
}
