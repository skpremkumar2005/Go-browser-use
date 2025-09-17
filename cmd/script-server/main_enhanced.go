package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"script-server/controller"
)

func main() {
	// Create Echo instance
	e := echo.New()

	// Enhanced middleware setup
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	
	// Custom middleware for WebSocket upgrade detection
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip middleware for WebSocket upgrade requests
			if c.Request().Header.Get("Upgrade") == "websocket" {
				log.Printf("⏭️ Skipping middleware for WebSocket upgrade: %s %s", c.Request().Method, c.Request().URL.Path)
				return next(c)
			}
			return next(c)
		}
	})

	// Initialize enhanced WebSocket handler
	enhancedHandler := controller.NewEnhancedWebSocketHandler()
	
	// Initialize session manager (if you have one)
	sessionManager := controller.NewSessionManager()

	// API Routes
	api := e.Group("/api/v1/browser_use")
	
	// Session management routes
	api.GET("/browser/active-session", func(c echo.Context) error {
		// Return active session info (implement based on your session management)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"sessionId": "session_1758121239539061900", // Replace with actual session logic
			"status":    "ready",
			"timestamp": 1758121239539061900,
		})
	})

	// Enhanced WebSocket routes - using the new enhanced handler
	api.GET("/browser/websocket-stream/:sessionId", enhancedHandler.HandleWebSocketEnhanced)
	api.GET("/browser/ws/:sessionId", enhancedHandler.HandleWebSocketEnhanced)

	// Health check route
	e.GET("/health", func(c echo.Context) error {
		sessionCount := enhancedHandler.GetSessionCount()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":           "healthy",
			"service":          "Enhanced Go WebSocket Script Server",
			"version":          "2.0.0",
			"active_sessions":  sessionCount,
			"timestamp":        1758121239539061900,
		})
	})

	// Static files
	e.Static("/", "../../frontend")

	// Start server info
	log.Printf("🚀 Starting Enhanced Go WebSocket Script Server")
	log.Printf("🌐 Server will be available at: http://localhost:3000")
	log.Printf("🔌 Enhanced WebSocket endpoints:")
	log.Printf("   • ws://localhost:3000/api/v1/browser_use/browser/websocket-stream/{sessionId}")
	log.Printf("   • ws://localhost:3000/api/v1/browser_use/browser/ws/{sessionId}")
	log.Printf("📊 Health check: http://localhost:3000/health")
	log.Printf("🎯 Frontend: http://localhost:3000/live_websocket.html")

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("🛑 Shutting down Enhanced Go WebSocket Script Server...")
		
		// Cleanup sessions
		log.Printf("🧹 Cleaning up %d active sessions", enhancedHandler.GetSessionCount())
		
		if err := e.Shutdown(nil); err != nil {
			log.Printf("❌ Server shutdown error: %v", err)
		}
		log.Println("✅ Enhanced server shutdown complete")
	}()

	// Start server
	port := ":3000"
	log.Printf("🎯 Enhanced server starting on port %s", port)
	
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Failed to start enhanced server: %v", err)
	}
}