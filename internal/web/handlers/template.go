package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/shindakun/bskyarchive/internal/models"
)

// TemplateData holds common data passed to templates
type TemplateData struct {
	Error   string
	Message string
	Handle  string // For login form - repopulates handle after validation errors
	Session interface{}
	Status  *models.ArchiveStatus
	Posts   []models.Post
	Media   map[string][]models.Media // Map of post URI to media items
	ParentPostsInArchive map[string]bool // Map of parent URIs that exist in local archive
	Profiles map[string]string // Map of DID to handle
	Exports []models.ExportRecord // List of exports for export management page
	Query   string
	Page    int
	Total   int
	PageSize int
	TotalPages int
	HasActiveOperation bool
	ShowAll bool // Show all posts from all users
	Version string // Application version
	CSRFToken string // CSRF token for forms and HTMX requests
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
		"sanitizeID": func(id string) string {
			// Replace special characters with hyphens to create valid CSS selectors
			// Replaces : / and any other non-alphanumeric characters
			replacer := strings.NewReplacer(
				":", "-",
				"/", "-",
				" ", "-",
			)
			sanitized := replacer.Replace(id)
			// Remove any remaining non-alphanumeric characters except hyphens
			var result strings.Builder
			for _, ch := range sanitized {
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
					result.WriteRune(ch)
				}
			}
			return result.String()
		},
	}
}

// renderTemplate renders a template with the base layout
func (h *Handlers) renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data TemplateData) error {
	// Add CSRF token to template data
	data.CSRFToken = csrf.Token(r)

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
