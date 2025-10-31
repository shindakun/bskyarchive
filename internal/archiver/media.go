package archiver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// DownloadMediaResult represents the result of a media download
type DownloadMediaResult struct {
	Media    models.Media
	Skipped  bool   // True if file already exists
	FilePath string // Local file path
}

// DownloadMedia downloads media from a URL and stores it with SHA-256 hash-based path
func DownloadMedia(mediaPath, url, postURI, mimeType, altText string, width, height int) (*DownloadMediaResult, error) {
	// Create HTTP request with User-Agent to avoid bot detection
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Download the media
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download media: status %d", resp.StatusCode)
	}

	// Get MIME type from response header if not provided
	if mimeType == "" {
		mimeType = resp.Header.Get("Content-Type")
	}

	// Read the content and calculate hash
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read media content: %w", err)
	}

	// Detect actual content type from magic bytes if we got HTML (anti-bot page)
	if len(content) > 0 && (mimeType == "" || strings.Contains(mimeType, "text/html")) {
		mimeType = http.DetectContentType(content)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Determine file extension from MIME type or URL
	ext := getFileExtension(mimeType, url)

	// Create content-addressable path: media/<hash[:2]>/<hash[2:4]>/<hash>.<ext>
	// This distributes files across directories to avoid too many files in one dir
	dir1 := hashStr[:2]
	dir2 := hashStr[2:4]
	dirPath := filepath.Join(mediaPath, dir1, dir2)

	// Create directories if they don't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create media directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(dirPath, hashStr+ext)

	// Check if file already exists (dedupe via content-addressable storage)
	if _, err := os.Stat(filePath); err == nil {
		// File exists, skip download
		media := models.Media{
			Hash:      hashStr,
			PostURI:   postURI,
			MimeType:  mimeType,
			FilePath:  filePath,
			SizeBytes: int64(len(content)),
			Width:     width,
			Height:    height,
			AltText:   altText,
			CreatedAt: time.Now(),
		}
		return &DownloadMediaResult{
			Media:    media,
			Skipped:  true,
			FilePath: filePath,
		}, nil
	}

	// Write content to file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write media file: %w", err)
	}

	// Create media record
	media := models.Media{
		Hash:      hashStr,
		PostURI:   postURI,
		MimeType:  mimeType,
		FilePath:  filePath,
		SizeBytes: int64(len(content)),
		Width:     width,
		Height:    height,
		AltText:   altText,
		CreatedAt: time.Now(),
	}

	return &DownloadMediaResult{
		Media:    media,
		Skipped:  false,
		FilePath: filePath,
	}, nil
}

// getFileExtension determines the file extension from MIME type or URL
func getFileExtension(mimeType, url string) string {
	// Try to get extension from MIME type first
	switch {
	case strings.HasPrefix(mimeType, "image/jpeg"), strings.HasPrefix(mimeType, "image/jpg"):
		return ".jpg"
	case strings.HasPrefix(mimeType, "image/png"):
		return ".png"
	case strings.HasPrefix(mimeType, "image/gif"):
		return ".gif"
	case strings.HasPrefix(mimeType, "image/webp"):
		return ".webp"
	case strings.HasPrefix(mimeType, "video/mp4"):
		return ".mp4"
	case strings.HasPrefix(mimeType, "video/webm"):
		return ".webm"
	}

	// Fall back to URL extension
	if idx := strings.LastIndex(url, "."); idx != -1 {
		ext := url[idx:]
		// Only return if it looks like a valid extension (< 6 chars, no slashes)
		if len(ext) < 6 && !strings.Contains(ext, "/") {
			return ext
		}
	}

	// Default to .bin if we can't determine
	return ".bin"
}
