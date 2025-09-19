package browser

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all HTTP routes for the browser controller
func SetupRoutes(router *mux.Router) {
	// Task management - matching original routes
	router.HandleFunc("/execute", CreateScriptTaskHandler).Methods("POST")
	router.HandleFunc("/result/{taskId}", GetTaskResultHandler).Methods("GET")
	router.HandleFunc("/status/{taskId}", GetTaskStatusHandler).Methods("GET")
	router.HandleFunc("/task-status/{taskId}", GetTaskExecutionStatusHandler).Methods("GET")
	router.HandleFunc("/stop/{taskId}", StopTaskHandler).Methods("POST")
	router.HandleFunc("/pause/{taskId}", PauseTaskHandler).Methods("POST")
	router.HandleFunc("/resume/{taskId}", ResumeTaskHandler).Methods("POST")

	//
	router.HandleFunc("/task/{sessionId}", GetScriptTaskHandler).Methods("GET")
	router.HandleFunc("/task/{sessionId}/cancel", CancelScriptTaskHandler).Methods("POST")
	router.HandleFunc("/sessions", ListScriptSessionsHandler).Methods("GET")

	// Live automation and streaming routes
	router.HandleFunc("/live-automation/{sessionId}", LiveAutomationHandler).Methods("GET")
	router.HandleFunc("/stream-screencast/{sessionId}", StreamScreencastHandler).Methods("GET")
	router.HandleFunc("/stream-screenshots/{sessionId}", StreamScriptScreenshotsHandler).Methods("GET")
	router.HandleFunc("/reset-stream/{sessionId}", ResetStreamHandler).Methods("POST")

	// WebSocket routes
	router.HandleFunc("/ws", HandleWebSocket)
	router.HandleFunc("/stream-ws/{sessionId}", HandleStreamWebSocket)

	// System status route
	router.HandleFunc("/status", StatusHandler).Methods("GET")
}

// CORS middleware to handle cross-origin requests
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// middleware moved to main.go
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
					// Note: Actual logging may be handled in main.go middleware
					lastTaskStatusLog[r.URL.Path] = time.Now()
				}
				taskStatusMutex.Unlock()
				
				// Periodic cleanup to prevent memory leak
				go cleanupTaskStatusLog()
			}
		}
		// Main middleware logic moved to main.go
		next.ServeHTTP(w, r)
	})
}
