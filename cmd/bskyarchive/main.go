package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/shindakun/bskyarchive/internal/archiver"
	"github.com/shindakun/bskyarchive/internal/auth"
	"github.com/shindakun/bskyarchive/internal/config"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/web/handlers"
	webmiddleware "github.com/shindakun/bskyarchive/internal/web/middleware"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[bskyarchive] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Bluesky Archive Tool...")

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}
	logger.Println("Configuration loaded successfully")

	// Initialize database
	logger.Printf("Initializing database at: %s", cfg.Archive.DBPath)
	db, err := storage.InitDB(cfg.Archive.DBPath)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	logger.Println("Database initialized successfully")

	// Initialize session manager with cookie security configuration
	// Determine cookie Secure flag based on configuration and BASE_URL
	cookieSecure := false
	switch cfg.OAuth.CookieSecure {
	case "true":
		cookieSecure = true
	case "auto":
		cookieSecure = cfg.IsHTTPS()
	case "false":
		cookieSecure = false
	}

	// Parse SameSite mode
	var sameSiteMode http.SameSite
	switch cfg.OAuth.CookieSameSite {
	case "strict":
		sameSiteMode = http.SameSiteStrictMode
	case "lax":
		sameSiteMode = http.SameSiteLaxMode
	case "none":
		sameSiteMode = http.SameSiteNoneMode
	default:
		sameSiteMode = http.SameSiteLaxMode // Default to Lax for OAuth compatibility
	}

	sessionManager := auth.InitSessions(
		cfg.OAuth.SessionSecret,
		cfg.OAuth.SessionMaxAge,
		cookieSecure,
		sameSiteMode,
		db,
	)
	logger.Printf("Session manager initialized (Secure=%v, SameSite=%v, MaxAge=%ds)",
		cookieSecure, sameSiteMode, cfg.OAuth.SessionMaxAge)

	// Initialize OAuth manager
	baseURL := cfg.GetBaseURL()
	oauthManager := auth.InitOAuth(baseURL, cfg.OAuth.Scopes, sessionManager)
	logger.Printf("OAuth manager initialized with base URL: %s", baseURL)
	logger.Printf("OAuth scopes: %v", cfg.OAuth.Scopes)

	// Initialize router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(webmiddleware.SecurityHeaders(cfg)) // Security headers on all responses
	r.Use(webmiddleware.MaxBytesMiddleware(cfg.Server.Security.MaxRequestBytes)) // Request size limit
	r.Use(webmiddleware.LoggingMiddleware(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	logger.Printf("Security headers enabled (HSTS=%v)", cfg.IsHTTPS())
	logger.Printf("Request size limit: %d bytes (%.1f MB)", cfg.Server.Security.MaxRequestBytes, float64(cfg.Server.Security.MaxRequestBytes)/(1024*1024))

	// CSRF protection (if enabled)
	if cfg.Server.Security.CSRFEnabled {
		csrfMiddleware := webmiddleware.CSRFProtection(
			[]byte(cfg.OAuth.SessionSecret),
			cookieSecure, // Same secure setting as cookies
		)
		r.Use(csrfMiddleware)
		logger.Printf("CSRF protection enabled (Secure=%v, FieldName=%s)",
			cookieSecure, cfg.Server.Security.CSRFFieldName)
	} else {
		logger.Println("WARNING: CSRF protection is DISABLED")
	}

	// Initialize archiver worker with OAuth manager for bskyoauth session access
	worker := archiver.NewWorker(db, cfg.Archive.MediaPath, 300, 5*time.Minute, oauthManager)

	// Initialize handlers
	h := handlers.New(db, sessionManager, oauthManager, worker, logger)

	// Public routes
	r.Get("/", h.Landing)
	r.Get("/about", h.About)

	// OAuth client metadata (required by bskyoauth)
	r.Get("/client-metadata.json", oauthManager.ClientMetadataHandler())

	// OAuth callback (must be at root level to match redirect_uri)
	r.Get("/callback", h.Callback)

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.HandleFunc("/login", h.Login) // OAuth login - handles GET (form) and POST (handle submission), exempt from CSRF
		r.Get("/logout", h.Logout)
	})

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(webmiddleware.RequireAuth(sessionManager))
		r.Get("/dashboard", h.Dashboard)
		r.Get("/archive", h.Archive)
		r.Post("/archive/start", h.ArchiveStart)
		r.Get("/archive/status", h.ArchiveStatus)
		r.Get("/browse", h.Browse)
		r.Get("/export", h.ExportPage)
		r.Post("/export/start", h.StartExport)
		r.Get("/export/progress/{job_id}", h.ExportProgress)
		r.Get("/export/row/*", h.ExportRow)           // Get single export row as HTML fragment
		r.Get("/export/download/*", h.DownloadExport)
		r.Delete("/export/delete/*", h.DeleteExport) // DELETE requires CSRF token (automatically applied)
		r.Get("/media/{hash}", h.ServeMedia)
	})

	// Static files
	r.Get("/static/*", h.ServeStatic)

	// 404 handler (must be last)
	r.NotFound(h.NotFound)

	// HTTP server configuration
	srv := &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Printf("Server starting on http://%s", cfg.GetAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited successfully")
}
