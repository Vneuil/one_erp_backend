package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Server encapsulates the Fiber web application
type Server struct {
	App    *fiber.App
	Config *config.Config
	DB     *gorm.DB
	V1     fiber.Router
}

// NewServer creates a configured Fiber server
func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: middleware.ErrorHandler,
		// Uploads are capped at 10 MB of file (see shared/attachment); the rest is multipart overhead.
		BodyLimit: 12 << 20,
	})

	// Global middlewares
	app.Use(middleware.Recover())
	app.Use(middleware.RequestID())
	app.Use(middleware.CORS(cfg.Frontend.BaseURL))
	app.Use(middleware.Logger())

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		dbStatus := "connected"
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}

		return response.OK(c, "Server is healthy", fiber.Map{
			"app":       cfg.App.Name,
			"env":       cfg.App.Env,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"database":  dbStatus,
		})
	})

	// API v1 route group
	v1 := app.Group("/api/v1")

	return &Server{
		App:    app,
		Config: cfg,
		DB:     db,
		V1:     v1,
	}
}

// Start runs the HTTP server and handles graceful shutdown on interrupt signals
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.Config.App.Port)

	// Channel to listen for errors coming from the listener
	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("starting HTTP server", "address", addr, "env", s.Config.App.Env)
		if err := s.App.Listen(addr); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Channel to listen for an interrupt or terminate signal from the OS
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		slog.Info("shutdown signal received", "signal", sig.String())

		// Graceful shutdown with 10 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.App.ShutdownWithContext(ctx); err != nil {
			slog.Error("failed to gracefully shutdown Fiber app", "error", err)
			return err
		}

		slog.Info("server shutdown completed successfully")
	}

	return nil
}
