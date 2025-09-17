package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/controller/browser"
)

func main() {
	log.Printf("🚀 Starting Go Browser Use Server...")

	// Load configuration
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("❌ Configuration validation failed: %v", err)
	}

	// Initialize browser components
	browser.InitializeGlobalVariables(cfg)
	
	// Reset streaming flags on startup
	browser.ResetAllStreamingFlags()

	// Start WebSocket manager
	browser.StartWebSocketManager()

	// Start health monitoring
	browser.StartHealthMonitoring()

	// Setup HTTP router
	router := mux.NewRouter()
	
	// Setup routes
	browser.SetupRoutes(router)

	// Apply middleware
	router.Use(browser.EnableCORS)
	router.Use(browser.LoggingMiddleware)

	// Configure HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("🌐 Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("📊 Max concurrent sessions: %d", cfg.Browser.MaxConcurrentSessions)
	log.Printf("🔧 Browser: %dx%d, Display: %s", cfg.Browser.Width, cfg.Browser.Height, cfg.Browser.Display)
	log.Printf("🐍 Python: %s, Scripts: %s", cfg.Script.PythonCommand, cfg.Script.ScriptDir)
	log.Printf("📡 CDP: %s:%v, Timeout: %v", cfg.CDP.Host, cfg.CDP.Ports, cfg.CDP.Timeout)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
