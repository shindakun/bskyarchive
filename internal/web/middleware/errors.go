package middleware

import (
	"html/template"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/shindakun/bskyarchive/internal/auth"
)

// ErrorHandler wraps an http.Handler and recovers from panics, rendering error pages
func ErrorHandler(next http.Handler, templates *template.Template, sessionManager *auth.SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[ERROR] Panic recovered: %v\n%s", err, debug.Stack())

				// Get session for error page context
				session, _ := sessionManager.GetSession(r)

				data := map[string]interface{}{
					"Session": session,
					"Error":   err,
				}

				w.WriteHeader(http.StatusInternalServerError)
				if renderErr := templates.ExecuteTemplate(w, "base.html", data); renderErr != nil {
					log.Printf("[ERROR] Failed to render 500 page: %v", renderErr)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// NotFoundHandler returns a handler for 404 errors
func NotFoundHandler(templates *template.Template, sessionManager *auth.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := sessionManager.GetSession(r)

		data := map[string]interface{}{
			"Session": session,
		}

		w.WriteHeader(http.StatusNotFound)
		if err := templates.ExecuteTemplate(w, "base.html", data); err != nil {
			log.Printf("[ERROR] Failed to render 404 page: %v", err)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}
}

// UnauthorizedHandler returns a handler for 401 errors
func UnauthorizedHandler(templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Session": nil,
		}

		w.WriteHeader(http.StatusUnauthorized)
		if err := templates.ExecuteTemplate(w, "base.html", data); err != nil {
			log.Printf("[ERROR] Failed to render 401 page: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}
}
