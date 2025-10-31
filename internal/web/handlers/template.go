package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/shindakun/bskyarchive/internal/models"
)

// TemplateData holds common data passed to templates
type TemplateData struct {
	Error   string
	Message string
	Session interface{}
	Status  *models.ArchiveStatus
	Posts   []models.Post
	Media   map[string][]models.Media // Map of post URI to media items
	ParentPostsInArchive map[string]bool // Map of parent URIs that exist in local archive
	Profiles map[string]string // Map of DID to handle
	Query   string
	Page    int
	Total   int
	PageSize int
	TotalPages int
	HasActiveOperation bool
	ShowAll bool // Show all posts from all users
	Version string // Application version
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"dec": func(i int) int {
			return i - 1
		},
		"extractPostID": func(uri string) string {
			// Extract post ID from AT URI
			// Format: at://did:plc:xxx/app.bsky.feed.post/xxxxx
			parts := strings.Split(uri, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
			return ""
		},
		"extractDID": func(uri string) string {
			// Extract DID from AT URI
			// Format: at://did:plc:xxx/app.bsky.feed.post/xxxxx
			if strings.HasPrefix(uri, "at://") {
				uri = strings.TrimPrefix(uri, "at://")
				parts := strings.Split(uri, "/")
				if len(parts) > 0 {
					return parts[0]
				}
			}
			return ""
		},
		"isValidImage": func(filePath string) bool {
			// Check if file has a valid image extension
			ext := strings.ToLower(filepath.Ext(filePath))
			validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
			for _, validExt := range validExts {
				if ext == validExt {
					return true
				}
			}
			return false
		},
	}
}

// renderTemplate renders a template with the base layout
func (h *Handlers) renderTemplate(w http.ResponseWriter, templateName string, data TemplateData) error {
	// Parse templates with functions - include partials for templates that need them
	files := []string{
		filepath.Join("internal", "web", "templates", "layouts", "base.html"),
		filepath.Join("internal", "web", "templates", "pages", templateName+".html"),
	}

	// Add partials directory for templates that use them
	partialsGlob := filepath.Join("internal", "web", "templates", "partials", "*.html")
	partials, _ := filepath.Glob(partialsGlob)
	files = append(files, partials...)

	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFiles(files...)
	if err != nil {
		return err
	}

	// Execute template
	return tmpl.ExecuteTemplate(w, "base", data)
}

// renderPartial renders a partial template (for HTMX)
func (h *Handlers) renderPartial(w http.ResponseWriter, partialName string, data TemplateData) error {
	// Parse partial template with functions
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFiles(
		filepath.Join("internal", "web", "templates", "partials", partialName+".html"),
	)
	if err != nil {
		return err
	}

	// Execute template
	return tmpl.ExecuteTemplate(w, partialName, data)
}
