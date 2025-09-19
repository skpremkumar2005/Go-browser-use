package browser

import (
	"net/http"

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

// Logging middleware to log HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// middleware moved to main.go
		next.ServeHTTP(w, r)
	})
}
