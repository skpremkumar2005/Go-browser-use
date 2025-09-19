package main

import (
	"log"
	"net/http"

	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/controller/browser"
	"go-webrtc/cmd/script-server/libs/utils/helper"

	"github.com/gorilla/mux"
)

func main() {
	log.Printf("🚀 Starting Go Browser Use Server...")

	// Load configuration
	cfg := config.GlobalEnv
	if err := helper.ValidateConfig(cfg); err != nil {
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
		Addr:         cfg["Server"].(map[string]interface{})["Host"].(string) + ":" + cfg["Server"].(map[string]interface{})["Port"].(string),
		Handler:      router,
		ReadTimeout:  helper.ParseDurationWithDefault("", cfg["Server"].(map[string]interface{})["ReadTimeout"].(string)),
		WriteTimeout: helper.ParseDurationWithDefault("", cfg["Server"].(map[string]interface{})["WriteTimeout"].(string)),
	}

	log.Printf("🌐 Server starting on %s:%s", cfg["Server"].(map[string]interface{})["Host"].(string), cfg["Server"].(map[string]interface{})["Port"].(string))
	log.Printf("📊 Max concurrent sessions: %d", cfg["Browser"].(map[string]interface{})["MaxConcurrentSessions"].(int))
	log.Printf("🔧 Browser: %dx%d, Display: %s", cfg["Browser"].(map[string]interface{})["Width"].(int), cfg["Browser"].(map[string]interface{})["Height"].(int), cfg["Browser"].(map[string]interface{})["Display"].(string))
	log.Printf("🐍 Python: %s, Scripts: %s", cfg["Script"].(map[string]interface{})["PythonCommand"].(string), cfg["Script"].(map[string]interface{})["ScriptDir"].(string))
	log.Printf("📡 CDP: %s:%v, Timeout: %v", cfg["CDP"].(map[string]interface{})["Host"].(string), cfg["CDP"].(map[string]interface{})["Ports"], helper.ParseDurationWithDefault("", cfg["CDP"].(map[string]interface{})["Timeout"].(string)))

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
