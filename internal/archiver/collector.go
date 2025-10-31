package archiver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/shindakun/bskyarchive/internal/models"
)

// PostsResult represents a batch of fetched posts with pagination info
type PostsResult struct {
	Posts  []models.Post
	Cursor string
	Total  int
}

// ProfileResult represents a fetched profile
type ProfileResult struct {
	Profile models.Profile
}

// FetchPosts retrieves posts from an author's feed with pagination
func FetchPosts(ctx context.Context, client *ATProtoClient, actor, cursor string, limit int64) (*PostsResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default batch size
	}

	// Call app.bsky.feed.getAuthorFeed using DPoP-authenticated client
	output, err := bsky.FeedGetAuthorFeed(ctx, client.GetClient(), actor, cursor, "", false, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch author feed: %w", err)
	}

	// Convert feed view posts to our Post model
	var posts []models.Post
	for _, feedPost := range output.Feed {
		post, err := convertFeedViewPostToPost(feedPost)
		if err != nil {
			// Log error but continue processing other posts
			fmt.Printf("Warning: failed to convert post: %v\n", err)
			continue
		}
		posts = append(posts, *post)
	}

	// Handle cursor (may be nil)
	cursorStr := ""
	if output.Cursor != nil {
		cursorStr = *output.Cursor
	}

	return &PostsResult{
		Posts:  posts,
		Cursor: cursorStr,
		Total:  len(posts),
	}, nil
}

// convertFeedViewPostToPost converts a bsky.FeedDefs_FeedViewPost to our models.Post
func convertFeedViewPostToPost(feedPost *bsky.FeedDefs_FeedViewPost) (*models.Post, error) {
	if feedPost == nil || feedPost.Post == nil {
		return nil, fmt.Errorf("invalid feed post")
	}

	p := feedPost.Post
	post := &models.Post{
		URI:        p.Uri,
		CID:        p.Cid,
		DID:        p.Author.Did,
		IndexedAt:  time.Now(), // Use current time for indexed_at
		ArchivedAt: time.Now(),
	}

	// Handle pointer fields (may be nil)
	if p.LikeCount != nil {
		post.LikeCount = int(*p.LikeCount)
	}
	if p.RepostCount != nil {
		post.RepostCount = int(*p.RepostCount)
	}
	if p.ReplyCount != nil {
		post.ReplyCount = int(*p.ReplyCount)
	}

	// Parse record to get text and created_at
	// Record is CBOR-encoded, marshal to JSON first
	if p.Record != nil {
		recordJSON, err := json.Marshal(p.Record.Val)
		if err == nil {
			var recMap map[string]interface{}
			if err := json.Unmarshal(recordJSON, &recMap); err == nil {
				if text, ok := recMap["text"].(string); ok {
					post.Text = text
				}
				if createdAt, ok := recMap["createdAt"].(string); ok {
					t, err := time.Parse(time.RFC3339, createdAt)
					if err == nil {
						post.CreatedAt = t
					}
				}
				// Check if it's a reply
				if reply, ok := recMap["reply"].(map[string]interface{}); ok {
					post.IsReply = true
					if parent, ok := reply["parent"].(map[string]interface{}); ok {
						if uri, ok := parent["uri"].(string); ok {
							post.ReplyParent = uri
						}
					}
				}
			}
		}
	}

	// Check for media embeds
	if p.Embed != nil && p.Embed.EmbedImages_View != nil && len(p.Embed.EmbedImages_View.Images) > 0 {
		post.HasMedia = true
		post.EmbedType = "images"

		// Serialize embed data
		embedData, err := json.Marshal(p.Embed)
		if err == nil {
			post.EmbedData = embedData
		}
	} else if p.Embed != nil && p.Embed.EmbedExternal_View != nil {
		post.EmbedType = "external"
		embedData, err := json.Marshal(p.Embed)
		if err == nil {
			post.EmbedData = embedData
		}
		// Mark as having media if external embed has a thumbnail
		if p.Embed.EmbedExternal_View.External != nil && p.Embed.EmbedExternal_View.External.Thumb != nil {
			post.HasMedia = true
		}
	} else if p.Embed != nil && p.Embed.EmbedRecord_View != nil {
		post.EmbedType = "record"
		embedData, err := json.Marshal(p.Embed)
		if err == nil {
			post.EmbedData = embedData
		}
	} else if p.Embed != nil && p.Embed.EmbedRecordWithMedia_View != nil {
		post.HasMedia = true
		post.EmbedType = "record_with_media"
		embedData, err := json.Marshal(p.Embed)
		if err == nil {
			post.EmbedData = embedData
		}
	}

	// Serialize labels if present
	if len(p.Labels) > 0 {
		labels, err := json.Marshal(p.Labels)
		if err == nil {
			post.Labels = labels
		}
	}

	return post, nil
}

// FetchProfile retrieves an actor's profile
func FetchProfile(ctx context.Context, client *ATProtoClient, actor string) (*ProfileResult, error) {
	// Call app.bsky.actor.getProfile using DPoP-authenticated client
	output, err := bsky.ActorGetProfile(ctx, client.GetClient(), actor)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	profile := models.Profile{
		DID:        output.Did,
		Handle:     output.Handle,
		SnapshotAt: time.Now(),
	}

	// Handle optional pointer fields
	if output.DisplayName != nil {
		profile.DisplayName = *output.DisplayName
	}
	if output.Description != nil {
		profile.Description = *output.Description
	}
	if output.FollowersCount != nil {
		profile.FollowersCount = int(*output.FollowersCount)
	}
	if output.FollowsCount != nil {
		profile.FollowsCount = int(*output.FollowsCount)
	}
	if output.PostsCount != nil {
		profile.PostsCount = int(*output.PostsCount)
	}

	// Avatar and banner URLs (if present)
	if output.Avatar != nil {
		profile.AvatarURL = *output.Avatar
	}
	if output.Banner != nil {
		profile.BannerURL = *output.Banner
	}

	return &ProfileResult{
		Profile: profile,
	}, nil
}
