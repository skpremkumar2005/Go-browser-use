package main

import (
	"log"
	"net/http"

	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/controller"
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

	// Create API subrouter and pass to controllers
	api := router.PathPrefix("/api").Subrouter()
	controller.SetupRoutes(api)

	// Static file serving for frontend (any non-/api route)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))

	// Apply middleware (defined in this package)
	router.Use(EnableCORS)
	router.Use(LoggingMiddleware)

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

// EnableCORS is a middleware that sets CORS headers and handles preflight requests.
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from config
		allowedOrigins := helper.GetAllowedOrigins()

		origin := r.Header.Get("Origin")
		if origin != "" {
			// Check if origin is allowed
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs basic request metadata.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📡 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
