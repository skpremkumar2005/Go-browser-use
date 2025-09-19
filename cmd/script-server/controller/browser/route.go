package browser

import (
	"log"
	"net/http"

	"go-webrtc/cmd/script-server/libs/utils/helper"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all HTTP routes for the browser controller
func SetupRoutes(router *mux.Router) {
	// API routes
	api := router.PathPrefix("/api").Subrouter()
	
	// Task management - matching original routes
	api.HandleFunc("/browser_use/execute", CreateScriptTaskHandler).Methods("POST")
	api.HandleFunc("/browser_use/result/{taskId}", GetTaskResultHandler).Methods("GET")
	api.HandleFunc("/browser_use/status/{taskId}", GetTaskStatusHandler).Methods("GET")
	api.HandleFunc("/browser_use/stop/{taskId}", StopTaskHandler).Methods("POST")

	//
	api.HandleFunc("/task/{sessionId}", GetScriptTaskHandler).Methods("GET")
	api.HandleFunc("/task/{sessionId}/cancel", CancelScriptTaskHandler).Methods("POST")
	api.HandleFunc("/sessions", ListScriptSessionsHandler).Methods("GET")
	
	// Live automation and streaming routes
	api.HandleFunc("/live-automation/{sessionId}", LiveAutomationHandler).Methods("GET")
	api.HandleFunc("/stream-screencast/{sessionId}", StreamScreencastHandler).Methods("GET")
	api.HandleFunc("/stream-screenshots/{sessionId}", StreamScriptScreenshotsHandler).Methods("GET")
	api.HandleFunc("/reset-stream/{sessionId}", ResetStreamHandler).Methods("POST")
	
	// WebSocket routes
	api.HandleFunc("/ws", HandleWebSocket)
	api.HandleFunc("/stream-ws/{sessionId}", HandleStreamWebSocket)
	
	// System status route
	api.HandleFunc("/status", StatusHandler).Methods("GET")
	
	// Static file serving for frontend
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend/")))
}

// CORS middleware to handle cross-origin requests
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

// Logging middleware to log HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📡 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
