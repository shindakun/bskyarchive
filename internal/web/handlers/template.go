package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// TemplateData holds common data passed to templates
type TemplateData struct {
	Error   string
	Message string
	Session interface{}
}

// renderTemplate renders a template with the base layout
func (h *Handlers) renderTemplate(w http.ResponseWriter, templateName string, data TemplateData) error {
	// Parse templates
	tmpl, err := template.ParseFiles(
		filepath.Join("internal", "web", "templates", "layouts", "base.html"),
		filepath.Join("internal", "web", "templates", "pages", templateName+".html"),
	)
	if err != nil {
		return err
	}

	// Execute template
	return tmpl.Execute(w, data)
}
