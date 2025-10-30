package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/web/handlers"
	webmiddleware "github.com/shindakun/bskyarchive/internal/web/middleware"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[bskyarchive] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Bluesky Archive Tool...")

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/archive.db"
	}
	logger.Printf("Initializing database at: %s", dbPath)

	db, err := storage.InitDB(dbPath)
	if err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	logger.Println("Database initialized successfully")

	// Initialize router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Custom middleware
	sessionStore := webmiddleware.NewSessionStore()
	r.Use(webmiddleware.SessionMiddleware(sessionStore))

	// Initialize handlers
	h := handlers.New(db, sessionStore, logger)

	// Public routes
	r.Get("/", h.Landing)
	r.Get("/about", h.About)

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.Login)
		r.Get("/callback", h.Callback)
		r.Get("/logout", h.Logout)
	})

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(webmiddleware.RequireAuth)
		r.Get("/dashboard", h.Dashboard)
		r.Get("/archive", h.Archive)
		r.Post("/archive/start", h.ArchiveStart)
		r.Get("/archive/status", h.ArchiveStatus)
		r.Get("/browse", h.Browse)
		r.Get("/media/{hash}", h.ServeMedia)
	})

	// Static files
	r.Get("/static/*", h.ServeStatic)

	// HTTP server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Printf("Server starting on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited successfully")
}
