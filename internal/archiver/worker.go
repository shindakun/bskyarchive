package archiver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyoauth"
)

// BskySessionGetter defines the interface for getting bskyoauth sessions
type BskySessionGetter interface {
	GetBskySession(sessionID string) (*bskyoauth.Session, error)
}

// Worker manages background archive operations
type Worker struct {
	db                *sql.DB
	mediaPath         string
	rateLimiter       *RateLimiter
	bskySessionGetter BskySessionGetter
}

// NewWorker creates a new archive worker
func NewWorker(db *sql.DB, mediaPath string, requestsPerWindow int, windowDuration time.Duration, bskySessionGetter BskySessionGetter) *Worker {
	return &Worker{
		db:                db,
		mediaPath:         mediaPath,
		rateLimiter:       NewRateLimiter(requestsPerWindow, windowDuration, 10),
		bskySessionGetter: bskySessionGetter,
	}
}

// StartArchive initiates a new archive operation for a user
// bskyoauthSessionID is the session ID from bskyoauth library
func (w *Worker) StartArchive(ctx context.Context, did, bskyoauthSessionID string, operationType models.OperationType) (string, error) {
	// Check if there's already an active operation
	activeOp, err := storage.GetActiveOperation(w.db, did)
	if err != nil {
		return "", fmt.Errorf("failed to check active operations: %w", err)
	}

	if activeOp != nil {
		return "", fmt.Errorf("archive operation already in progress: %s", activeOp.ID)
	}

	// Create new operation
	operationID := uuid.New().String()
	operation := &models.ArchiveOperation{
		ID:              operationID,
		DID:             did,
		Type:            operationType,
		Status:          models.OperationStatusPending,
		ProgressCurrent: 0,
		ProgressTotal:   0, // Will be set once we know total posts
		StartedAt:       time.Now(),
	}

	if err := storage.CreateOperation(w.db, operation); err != nil {
		return "", fmt.Errorf("failed to create operation: %w", err)
	}

	// Launch worker goroutine
	go w.archiveWorker(context.Background(), operationID, did, bskyoauthSessionID)

	return operationID, nil
}

// archiveWorker runs the actual archive process in the background
func (w *Worker) archiveWorker(ctx context.Context, operationID, did, bskyoauthSessionID string) {
	log.Printf("Starting archive operation %s for DID %s", operationID, did)

	// Update operation status to running
	operation, err := storage.GetOperation(w.db, operationID)
	if err != nil {
		log.Printf("Failed to get operation: %v", err)
		return
	}

	operation.Status = models.OperationStatusRunning
	if err := storage.UpdateOperation(w.db, operation); err != nil {
		log.Printf("Failed to update operation status: %v", err)
		return
	}

	// Get the bskyoauth session with DPoP key
	bskySession, err := w.bskySessionGetter.GetBskySession(bskyoauthSessionID)
	if err != nil {
		log.Printf("Failed to get bskyoauth session: %v", err)
		operation.Status = models.OperationStatusFailed
		operation.ErrorMessage = fmt.Sprintf("failed to get session: %v", err)
		now := time.Now()
		operation.CompletedAt = &now
		_ = storage.UpdateOperation(w.db, operation)
		return
	}

	// Create AT Protocol client with DPoP authentication
	client, err := NewATProtoClientFromSession(ctx, bskySession)
	if err != nil {
		log.Printf("Failed to create AT Protocol client: %v", err)
		operation.Status = models.OperationStatusFailed
		operation.ErrorMessage = fmt.Sprintf("failed to create client: %v", err)
		now := time.Now()
		operation.CompletedAt = &now
		_ = storage.UpdateOperation(w.db, operation)
		return
	}

	// Fetch and save profile first
	if err := w.fetchProfile(ctx, client, did); err != nil {
		log.Printf("Warning: failed to fetch profile: %v", err)
		// Continue anyway - profile is not critical
	}

	// Fetch posts with pagination
	var cursor string
	totalPosts := 0
	batchSize := int64(50)

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			operation.Status = models.OperationStatusCancelled
			operation.ErrorMessage = "cancelled by user"
			now := time.Now()
			operation.CompletedAt = &now
			_ = storage.UpdateOperation(w.db, operation)
			return
		default:
		}

		// Rate limit
		if err := w.rateLimiter.Wait(ctx); err != nil {
			log.Printf("Rate limiter cancelled: %v", err)
			break
		}

		// Fetch batch of posts
		result, err := FetchPosts(ctx, client, did, cursor, batchSize)
		if err != nil {
			log.Printf("Failed to fetch posts: %v", err)
			operation.Status = models.OperationStatusFailed
			operation.ErrorMessage = fmt.Sprintf("failed to fetch posts: %v", err)
			now := time.Now()
			operation.CompletedAt = &now
			_ = storage.UpdateOperation(w.db, operation)
			return
		}

		// Process each post
		for _, post := range result.Posts {
			// Save post
			if err := storage.SavePost(w.db, &post); err != nil {
				log.Printf("Warning: failed to save post %s: %v", post.URI, err)
				continue
			}

			// Download media if present
			if post.HasMedia && post.EmbedData != nil {
				if err := w.downloadPostMedia(ctx, &post); err != nil {
					log.Printf("Warning: failed to download media for post %s: %v", post.URI, err)
				}
			}

			totalPosts++
		}

		// Update progress
		operation.ProgressCurrent = int64(totalPosts)
		if err := storage.UpdateOperation(w.db, operation); err != nil {
			log.Printf("Warning: failed to update progress: %v", err)
		}

		// Check if we have more pages
		if result.Cursor == "" || len(result.Posts) == 0 {
			break
		}

		cursor = result.Cursor
	}

	// Mark operation as completed
	operation.Status = models.OperationStatusCompleted
	operation.ProgressCurrent = int64(totalPosts)
	operation.ProgressTotal = int64(totalPosts)
	now := time.Now()
	operation.CompletedAt = &now

	if err := storage.UpdateOperation(w.db, operation); err != nil {
		log.Printf("Failed to mark operation as completed: %v", err)
	}

	log.Printf("Archive operation %s completed: %d posts archived", operationID, totalPosts)
}

// fetchProfile fetches and saves the user's profile
func (w *Worker) fetchProfile(ctx context.Context, client *ATProtoClient, did string) error {
	result, err := FetchProfile(ctx, client, did)
	if err != nil {
		return err
	}

	return storage.SaveProfile(w.db, &result.Profile)
}

// downloadPostMedia downloads all media from a post's embed data
func (w *Worker) downloadPostMedia(ctx context.Context, post *models.Post) error {
	// Parse embed data
	var embedData map[string]interface{}
	if err := json.Unmarshal(post.EmbedData, &embedData); err != nil {
		return fmt.Errorf("failed to parse embed data: %w", err)
	}

	// Handle different embed types
	if post.EmbedType == "images" || post.EmbedType == "record_with_media" {
		images, err := extractImages(embedData)
		if err != nil {
			return err
		}

		for _, img := range images {
			// Rate limit
			if err := w.rateLimiter.Wait(ctx); err != nil {
				return err
			}

			// Download image
			result, err := DownloadMedia(
				w.mediaPath,
				img.URL,
				post.URI,
				img.MimeType,
				img.AltText,
				img.Width,
				img.Height,
			)
			if err != nil {
				log.Printf("Warning: failed to download image: %v", err)
				continue
			}

			// Save media metadata
			if err := storage.SaveMedia(w.db, &result.Media); err != nil {
				log.Printf("Warning: failed to save media metadata: %v", err)
			}
		}
	} else if post.EmbedType == "external" {
		// Download thumbnail from external link embed
		thumbnail, err := extractExternalThumbnail(embedData)
		if err == nil && thumbnail.URL != "" {
			// Rate limit
			if err := w.rateLimiter.Wait(ctx); err != nil {
				return err
			}

			// Download thumbnail
			result, err := DownloadMedia(
				w.mediaPath,
				thumbnail.URL,
				post.URI,
				thumbnail.MimeType,
				thumbnail.AltText,
				thumbnail.Width,
				thumbnail.Height,
			)
			if err != nil {
				log.Printf("Warning: failed to download external thumbnail: %v", err)
			} else {
				// Save media metadata
				if err := storage.SaveMedia(w.db, &result.Media); err != nil {
					log.Printf("Warning: failed to save media metadata: %v", err)
				}
			}
		}

		// Also download the main external resource (GIF, video, etc.)
		resource, err := extractExternalResource(embedData)
		if err == nil && resource.URL != "" {
			// Rate limit
			if err := w.rateLimiter.Wait(ctx); err != nil {
				return err
			}

			// Download main resource
			result, err := DownloadMedia(
				w.mediaPath,
				resource.URL,
				post.URI,
				resource.MimeType,
				resource.AltText,
				resource.Width,
				resource.Height,
			)
			if err != nil {
				log.Printf("Warning: failed to download external resource: %v", err)
			} else {
				// Save media metadata
				if err := storage.SaveMedia(w.db, &result.Media); err != nil {
					log.Printf("Warning: failed to save media metadata: %v", err)
				}
			}
		}
	}

	return nil
}

// ImageInfo represents image metadata from embed data
type ImageInfo struct {
	URL      string
	MimeType string
	AltText  string
	Width    int
	Height   int
}

// extractImages extracts image information from embed data
func extractImages(embedData map[string]interface{}) ([]ImageInfo, error) {
	var images []ImageInfo

	// Look for images in EmbedImages_View
	if view, ok := embedData["EmbedImages_View"].(map[string]interface{}); ok {
		if imagesArray, ok := view["images"].([]interface{}); ok {
			for _, imgInterface := range imagesArray {
				if img, ok := imgInterface.(map[string]interface{}); ok {
					info := ImageInfo{}

					if fullsize, ok := img["fullsize"].(string); ok {
						info.URL = fullsize
					}
					if alt, ok := img["alt"].(string); ok {
						info.AltText = alt
					}

					// Try to get dimensions from aspectRatio if available
					if aspectRatio, ok := img["aspectRatio"].(map[string]interface{}); ok {
						if width, ok := aspectRatio["width"].(float64); ok {
							info.Width = int(width)
						}
						if height, ok := aspectRatio["height"].(float64); ok {
							info.Height = int(height)
						}
					}

					// Default MIME type for images
					info.MimeType = "image/jpeg"

					if info.URL != "" {
						images = append(images, info)
					}
				}
			}
		}
	}

	// Look for images in EmbedRecordWithMedia_View
	if media, ok := embedData["media"].(map[string]interface{}); ok {
		if mediaImages, ok := media["EmbedImages_View"].(map[string]interface{}); ok {
			if imagesArray, ok := mediaImages["images"].([]interface{}); ok {
				for _, imgInterface := range imagesArray {
					if img, ok := imgInterface.(map[string]interface{}); ok {
						info := ImageInfo{}

						if fullsize, ok := img["fullsize"].(string); ok {
							info.URL = fullsize
						}
						if alt, ok := img["alt"].(string); ok {
							info.AltText = alt
						}

						info.MimeType = "image/jpeg"

						if info.URL != "" {
							images = append(images, info)
						}
					}
				}
			}
		}
	}

	return images, nil
}

// extractExternalThumbnail extracts thumbnail information from external embed data
func extractExternalThumbnail(embedData map[string]interface{}) (ImageInfo, error) {
	var thumbnail ImageInfo

	// Look for external embed thumbnail - the structure is flat with "external" at top level
	if external, ok := embedData["external"].(map[string]interface{}); ok {
		// Get thumbnail URL
		if thumb, ok := external["thumb"].(string); ok {
			thumbnail.URL = thumb
		}

		// Get alt text from description
		if desc, ok := external["description"].(string); ok {
			thumbnail.AltText = desc
		}

		// Default MIME type for thumbnails (usually JPEG)
		thumbnail.MimeType = "image/jpeg"
	}

	return thumbnail, nil
}

// extractExternalResource extracts the main external resource (image/GIF/video URL)
func extractExternalResource(embedData map[string]interface{}) (ImageInfo, error) {
	var resource ImageInfo

	// Look for external embed main resource - the structure is flat with "external" at top level
	if external, ok := embedData["external"].(map[string]interface{}); ok {
		// Get the main resource URL
		if uri, ok := external["uri"].(string); ok {
			resource.URL = uri
		}

		// Get alt text from description
		if desc, ok := external["description"].(string); ok {
			resource.AltText = desc
		}

		// Try to determine MIME type from URL
		// Most external embeds are GIFs, images, or videos
		resource.MimeType = "" // Will be determined from URL extension
	}

	return resource, nil
}
