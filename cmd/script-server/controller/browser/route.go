package browser

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

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
	api.HandleFunc("/browser_use/task-status/{taskId}", GetTaskExecutionStatusHandler).Methods("GET")
	api.HandleFunc("/browser_use/stop/{taskId}", StopTaskHandler).Methods("POST")
	api.HandleFunc("/browser_use/pause/{taskId}", PauseTaskHandler).Methods("POST")
	api.HandleFunc("/browser_use/resume/{taskId}", ResumeTaskHandler).Methods("POST")

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

// Global variables for smart logging
var (
	lastTaskStatusLog = make(map[string]time.Time)
	taskStatusMutex   sync.RWMutex
	lastCleanup       time.Time
)

// Clean up old entries to prevent memory leak
func cleanupTaskStatusLog() {
	taskStatusMutex.Lock()
	defer taskStatusMutex.Unlock()
	
	// Only cleanup every 10 minutes
	if time.Since(lastCleanup) < 10*time.Minute {
		return
	}
	
	cutoff := time.Now().Add(-1 * time.Hour) // Remove entries older than 1 hour
	for path, lastTime := range lastTaskStatusLog {
		if lastTime.Before(cutoff) {
			delete(lastTaskStatusLog, path)
		}
	}
	lastCleanup = time.Now()
	log.Printf("🧹 Cleaned up task status log, %d entries remaining", len(lastTaskStatusLog))
}

// Smart logging middleware - reduces spam for high-frequency endpoints
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for high-frequency polling endpoints to reduce spam
		if strings.Contains(r.URL.Path, "/task-status/") {
			// Only log task-status requests every 30 seconds per endpoint
			taskStatusMutex.RLock()
			lastLog, exists := lastTaskStatusLog[r.URL.Path]
			shouldLog := !exists || time.Since(lastLog) > 30*time.Second
			taskStatusMutex.RUnlock()
			
			if shouldLog {
				taskStatusMutex.Lock()
				// Double-check after acquiring write lock
				if lastLog, exists := lastTaskStatusLog[r.URL.Path]; !exists || time.Since(lastLog) > 30*time.Second {
					log.Printf("📡 %s %s from %s (polling - reduced logging)", r.Method, r.URL.Path, r.RemoteAddr)
					lastTaskStatusLog[r.URL.Path] = time.Now()
				}
				taskStatusMutex.Unlock()
				
				// Periodic cleanup to prevent memory leak
				go cleanupTaskStatusLog()
			}
		} else {
			// Log all other requests normally
			log.Printf("📡 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		next.ServeHTTP(w, r)
	})
}
