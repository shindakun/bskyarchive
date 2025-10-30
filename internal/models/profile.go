package models

import (
	"fmt"
	"time"
)

// Profile represents a snapshot of a user's Bluesky profile at a specific point in time
type Profile struct {
	DID            string    `json:"did" db:"did"`
	Handle         string    `json:"handle" db:"handle"`
	DisplayName    string    `json:"display_name" db:"display_name"`
	Description    string    `json:"description" db:"description"`
	AvatarURL      string    `json:"avatar_url" db:"avatar_url"`
	BannerURL      string    `json:"banner_url" db:"banner_url"`
	FollowersCount int       `json:"followers_count" db:"followers_count"`
	FollowsCount   int       `json:"follows_count" db:"follows_count"`
	PostsCount     int       `json:"posts_count" db:"posts_count"`
	SnapshotAt     time.Time `json:"snapshot_at" db:"snapshot_at"`
}

// Validate checks if the profile fields are valid
func (p *Profile) Validate() error {
	if p.DID == "" {
		return fmt.Errorf("did is required")
	}

	if p.Handle == "" {
		return fmt.Errorf("handle is required")
	}

	if len(p.DisplayName) > 64 {
		return fmt.Errorf("display_name must be at most 64 characters")
	}

	if len(p.Description) > 256 {
		return fmt.Errorf("description must be at most 256 characters")
	}

	if p.FollowersCount < 0 {
		return fmt.Errorf("followers_count must be non-negative")
	}

	if p.FollowsCount < 0 {
		return fmt.Errorf("follows_count must be non-negative")
	}

	if p.PostsCount < 0 {
		return fmt.Errorf("posts_count must be non-negative")
	}

	return nil
}
