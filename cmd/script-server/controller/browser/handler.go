package browser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// HTTP Handlers

func CreateScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecuteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse := CreateSimpleErrorResponse("invalid-json", "Invalid JSON in request body", http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Validate required fields
	if req.Task == "" {
		errorResponse := CreateSimpleErrorResponse("missing-task", "Task field is required", http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	var browserSession *BrowserSession
	var sessionReused = false
	var taskID string

	// Check if we should reuse an existing browser session
	if req.SessionID != "" {
		log.Printf("🔄 Attempting to reuse browser session: %s", req.SessionID)
		browserSession = GetBrowserManager().GetSession(req.SessionID)
		
		if browserSession != nil {
			log.Printf("📊 Found browser session %s with status: %s", req.SessionID, browserSession.Status)
			if browserSession.Status == "ready" {
				log.Printf("✅ Reusing existing browser session: %s", req.SessionID)
				sessionReused = true
				taskID = req.SessionID // Use session ID as task ID for consistency
				
				// Cancel any existing cleanup timer
				GetBrowserManager().cancelCleanupTimer(req.SessionID)
			} else {
				log.Printf("⚠️ Browser session %s status is '%s', not 'ready' - cannot reuse", req.SessionID, browserSession.Status)
				browserSession = nil
			}
		} else {
			log.Printf("⚠️ Browser session %s not found", req.SessionID)
		}
	}

	// Create new browser session if not reusing
	if browserSession == nil {
		sessionID := fmt.Sprintf("browser_session_%d", time.Now().UnixNano())
		taskID = sessionID // Task ID = Session ID for new sessions too
		
		viewport := Viewport{
			Width:  GetConfig()["Browser"].(map[string]interface{})["ViewportWidth"].(int),
			Height: GetConfig()["Browser"].(map[string]interface{})["ViewportHeight"].(int),
		}

		var err error
		browserSession, err = GetBrowserManager().CreateBrowserSession(sessionID, viewport)
		if err != nil {
			// Check if it's a concurrency limit error
			if strings.Contains(err.Error(), "maximum concurrent sessions") {
				errorResponse := CreateSimpleErrorResponse("concurrency-limit-reached",
					fmt.Sprintf("Server is at capacity. Please try again later. %v", err),
					http.StatusServiceUnavailable)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(errorResponse)
			} else {
				errorResponse := CreateSimpleErrorResponse("browser-creation-failed",
					fmt.Sprintf("Failed to create browser session: %v", err),
					http.StatusInternalServerError)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(errorResponse)
			}
			return
		}
		
		log.Printf("✅ Created new browser session: %s", browserSession.ID)
	}

	// Check if task already exists (for session reuse)
	GetScriptSessionManager().mutex.RLock()
	existingTask := GetScriptSessionManager().sessions[taskID]
	GetScriptSessionManager().mutex.RUnlock()

	var scriptSession *ScriptSession

	if existingTask != nil {
		// Update existing task with new task details
		log.Printf("🔄 Updating existing task %s with new task: %s", taskID, req.Task)
		
		GetScriptSessionManager().mutex.Lock()
		existingTask.Task = req.Task
		existingTask.Status = "queued"
		existingTask.UpdatedAt = time.Now()
		
		// Update LLM config if provided
		if req.LLMModel != nil {
			existingTask.Metadata["llm_config"] = req.LLMModel
		}
		
		// Update maxSteps
		maxSteps := req.MaxSteps
		if maxSteps <= 0 {
			maxSteps = GetConfig()["Script"].(map[string]interface{})["MaxSteps"].(int)
		}
		existingTask.Metadata["max_steps"] = maxSteps
		GetScriptSessionManager().mutex.Unlock()
		
		scriptSession = existingTask
	} else {
		// Create new script session
		scriptSession = &ScriptSession{
			ID:          taskID,               // Task ID = Session ID
			Task:        req.Task,
			Status:      "queued",
			BrowserID:   browserSession.ID,    // Browser session ID
			CDPEndpoint: browserSession.CDPEndpoint,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		}

		// Store LLM config in metadata if provided
		if req.LLMModel != nil {
			scriptSession.Metadata["llm_config"] = req.LLMModel
		}

		// Store maxSteps in metadata
		maxSteps := req.MaxSteps
		if maxSteps <= 0 {
			maxSteps = GetConfig()["Script"].(map[string]interface{})["MaxSteps"].(int) // Use default from config
		}
		scriptSession.Metadata["max_steps"] = maxSteps

		GetScriptSessionManager().mutex.Lock()
		GetScriptSessionManager().sessions[taskID] = scriptSession
		GetScriptSessionManager().mutex.Unlock()
	}

	log.Printf("✅ Task %s in session %s: %s", taskID, browserSession.ID, req.Task)

	// Start task execution in background
	maxSteps := req.MaxSteps
	if maxSteps <= 0 {
		maxSteps = GetConfig()["Script"].(map[string]interface{})["MaxSteps"].(int)
	}
	go ExecuteScriptTask(taskID, req.Task, maxSteps)

	// Create simplified response - Task ID and Session ID are the same!
	response := CreateSimpleTaskResponse(scriptSession, browserSession, r, sessionReused)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GetScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		errorResponse := CreateErrorResponse("session-not-found", "Session not found", http.StatusNotFound, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Get browser session
	browserSession := GetBrowserManager().GetSession(session.BrowserID)

	// Create enhanced response
	response := CreateEnhancedTaskResponse(session, browserSession, r)

	// Update message based on status
	switch session.Status {
	case "queued":
		response.Message = "Task is queued and waiting to start"
	case "running":
		response.Message = "Task is currently running"
	case "completed":
		response.Message = "Task completed successfully"
	case "failed":
		response.Message = "Task failed to complete"
		response.Success = false
		if errorData, exists := session.Metadata["error"]; exists {
			if errorMsg, ok := errorData.(string); ok {
				response.Error = &ErrorInfo{
					Type:    "task-execution-failed",
					Message: errorMsg,
				}
			}
		}
	case "cancelled":
		response.Message = "Task was cancelled"
		response.Success = false
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func ListScriptSessionsHandler(w http.ResponseWriter, r *http.Request) {
	GetScriptSessionManager().mutex.RLock()
	sessions := make([]*ScriptSession, 0, len(GetScriptSessionManager().sessions))
	for _, session := range GetScriptSessionManager().sessions {
		sessions = append(sessions, session)
	}
	GetScriptSessionManager().mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func CancelScriptTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	GetScriptSessionManager().mutex.Lock()
	session := GetScriptSessionManager().sessions[sessionID]
	if session != nil {
		session.Status = "cancelled"
		session.UpdatedAt = time.Now()
		if session.Process != nil {
			session.Process.Process.Kill()
		}
	}
	GetScriptSessionManager().mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// Live automation handler - starts browser-use automation and shows live stream
func LiveAutomationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session information
	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session information
	browserSession := GetBrowserManager().GetSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Check if automation is already running
	if session.Status == "running" {
		log.Printf("🤖 Automation already running for session: %s", sessionID)
	} else if session.Status == "queued" || session.Status == "completed" || session.Status == "failed" {
		// Validate browser session is ready before starting automation
		if browserSession.BrowserProcess == nil {
			log.Printf("❌ Browser process is nil, cannot start automation for session: %s", sessionID)
			http.Error(w, "Browser process not ready", http.StatusInternalServerError)
			return
		}

		// Check if browser process is still running
		if browserSession.BrowserProcess.ProcessState != nil && browserSession.BrowserProcess.ProcessState.Exited() {
			log.Printf("❌ Browser process has exited, cannot start automation for session: %s", sessionID)
			http.Error(w, "Browser process has exited", http.StatusInternalServerError)
			return
		}

		// Test CDP endpoint before starting automation
		log.Printf("🔍 Testing CDP endpoint before starting automation: %s", browserSession.CDPEndpoint)
		testClient := NewCDPClient(browserSession.CDPEndpoint)
		if testErr := testClient.Connect(); testErr != nil {
			log.Printf("❌ CDP endpoint not accessible, cannot start automation: %v", testErr)
			testClient.Close()
			http.Error(w, fmt.Sprintf("Browser not accessible: %v", testErr), http.StatusInternalServerError)
			return
		}
		testClient.Close()
		log.Printf("✅ CDP endpoint validated, browser is ready for automation")

		// Start browser-use automation
		log.Printf("🤖 Starting browser-use automation for session: %s", sessionID)

		// Update status to running
		GetScriptSessionManager().mutex.Lock()
		session.Status = "running"
		session.UpdatedAt = time.Now()
		GetScriptSessionManager().mutex.Unlock()

		// Start automation in background with a small delay to ensure browser is ready
		go func() {
			time.Sleep(2 * time.Second)                    // Give browser time to fully initialize
			ExecuteScriptTask(sessionID, session.Task, 10) // Default max steps
		}()
	}

	// Prepare template data
	templateData := struct {
		TaskID       string
		Task         string
		Status       string
		CreatedAt    string
		BrowserID    string
		IsAutomation bool
		// Added: viewport dimensions for accurate client-side coordinate scaling
		ViewportWidth  int
		ViewportHeight int
	}{
		TaskID:       session.ID,
		Task:         session.Task,
		Status:       session.Status,
		CreatedAt:    session.CreatedAt.Format("2006-01-02 15:04:05"),
		BrowserID:    session.BrowserID,
		IsAutomation: true, // Flag to indicate this is automation mode
		ViewportWidth:  browserSession.Viewport.Width,
		ViewportHeight: browserSession.Viewport.Height,
	}

	// Read the live.html template
	templatePath := "frontend/live.html"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		// Try alternative paths
		altPaths := []string{"../frontend/live.html", "../../frontend/live.html"}
		for _, path := range altPaths {
			if _, err := os.Stat(path); err == nil {
				templatePath = path
				break
			}
		}
	}

	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Parse and execute template
	tmpl, err := template.New("live").Parse(string(templateContent))
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, templateData)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}

	log.Printf("🤖 Served live automation page for session: %s", sessionID)
}

// New screencast streaming handler (similar to unified-browser-platform)
func StreamScreencastHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := GetBrowserManager().GetSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Check if already streaming to prevent multiple concurrent streams
	if browserSession.Streaming {
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		http.Error(w, "Stream already active", http.StatusConflict)
		return
	}

	// Mark as streaming
	browserSession.Streaming = true
	defer func() {
		browserSession.Streaming = false
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	// Note: Streaming flag will be reset by defer function or manual reset endpoint

	log.Printf("📹 Starting screenshot stream for session %s, browser %s, CDP: %s", sessionID, session.BrowserID, browserSession.CDPEndpoint)

	// Set up MJPEG streaming
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create CDP client with retry logic
	var cdpClient *CDPClient
	var connectErr error
	maxConnectRetries := 3

	for attempt := 1; attempt <= maxConnectRetries; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/%d)", attempt, maxConnectRetries)

		cdpClient = NewCDPClient(browserSession.CDPEndpoint)
		connectErr = cdpClient.Connect()

		if connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}

		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)
		cdpClient.Close()

		if attempt < maxConnectRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after %d attempts: %v", maxConnectRetries, connectErr)
		http.Error(w, fmt.Sprintf("Failed to connect to browser: %v", connectErr), http.StatusInternalServerError)
		return
	}
	defer cdpClient.Close()

	// Add timeout for streaming (10 minutes max for automation)
	timeout := time.After(10 * time.Minute)
	frameCount := 0

	for {
		select {
		case <-time.After(100 * time.Millisecond): // 10 FPS
			// Check if task is completed
			GetScriptSessionManager().mutex.RLock()
			currentStatus := session.Status
			GetScriptSessionManager().mutex.RUnlock()

			// Stop streaming when task fails
			if currentStatus == "failed" {
				log.Printf("📸 Stopping stream for session %s (task failed)", sessionID)
				return
			}

			// Stop streaming when task completes
			if currentStatus == "completed" {
				log.Printf("📸 Stopping stream for session %s (task completed)", sessionID)
				return
			}

			// Note: We don't stop streaming when paused - streaming continues during pause

			// Capture screenshot with retry logic
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				log.Printf("⚠️ Failed to capture screenshot after retries: %v", err)
				// Try to reconnect CDP client
				if reconnectErr := cdpClient.Reconnect(); reconnectErr != nil {
					log.Printf("❌ Failed to reconnect CDP client: %v", reconnectErr)
					// Skip this frame, keep last image visible
					continue
				}
				log.Printf("✅ CDP client reconnected successfully")
				// Skip this frame, will try again next iteration
				continue
			}

			// Validate image data - skip if too small
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), likely blank, skipping frame", len(imageData))
				continue
			}

			// Send frame
			frameCount++
			if frameCount%50 == 0 {
				log.Printf("📹 Stream healthy: %d frames captured successfully", frameCount)
			}

			// Write MJPEG frame
			w.Write([]byte("--frame\r\n"))
			w.Write([]byte("Content-Type: image/jpeg\r\n"))
			w.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(imageData))))
			w.Write(imageData)
			w.Write([]byte("\r\n"))

			// Check if client disconnected
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Note: Stream health is monitored by frontend, not backend

		case <-timeout:
			log.Printf("📸 Streaming timeout for session %s", sessionID)
			return

		case <-r.Context().Done():
			log.Printf("📸 Client disconnected for session %s", sessionID)
			// Small delay to ensure proper cleanup
			time.Sleep(100 * time.Millisecond)
			return
		}
	}
}

// Reset stream handler for debugging
func ResetStreamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := GetBrowserManager().GetSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Reset streaming flag
	browserSession.Streaming = false
	log.Printf("🔄 Manually reset streaming flag for session %s", sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Streaming flag reset",
		"sessionId": sessionID,
	})
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// Get system status
	GetBrowserManager().mutex.RLock()
	activeBrowserSessions := len(GetBrowserManager().sessions)
	GetBrowserManager().mutex.RUnlock()

	GetScriptSessionManager().mutex.RLock()
	activeScriptSessions := len(GetScriptSessionManager().sessions)
	GetScriptSessionManager().mutex.RUnlock()

	GetBrowserManager().sessionMutex.Lock()
	concurrentSessions := GetBrowserManager().activeSessions
	maxSessions := GetBrowserManager().maxConcurrentSessions
	GetBrowserManager().sessionMutex.Unlock()

	// Get memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := map[string]interface{}{
		"status": "healthy",
		"concurrency": map[string]interface{}{
			"active_sessions":     concurrentSessions,
			"max_sessions":        maxSessions,
			"utilization_percent": float64(concurrentSessions) / float64(maxSessions) * 100,
		},
		"sessions": map[string]interface{}{
			"browser_sessions": activeBrowserSessions,
			"script_sessions":  activeScriptSessions,
		},
		"memory": map[string]interface{}{
			"alloc_mb": memStats.Alloc / 1024 / 1024,
			"sys_mb":   memStats.Sys / 1024 / 1024,
			"num_gc":   memStats.NumGC,
		},
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StreamScriptScreenshotsHandler streams screenshots during script execution (MJPEG format)
func StreamScriptScreenshotsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := GetBrowserManager().GetSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Check if already streaming to prevent multiple concurrent streams
	if browserSession.Streaming {
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		http.Error(w, "Stream already active", http.StatusConflict)
		return
	}

	// Mark as streaming
	browserSession.Streaming = true
	defer func() {
		browserSession.Streaming = false
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	log.Printf("📹 Starting stream for session %s, browser %s, CDP: %s", sessionID, session.BrowserID, browserSession.CDPEndpoint)

	// Set up MJPEG streaming
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Check browser health before starting stream
	if err := GetBrowserManager().checkSessionHealth(session.BrowserID); err != nil {
		log.Printf("❌ Browser health check failed for session %s: %v", session.BrowserID, err)

		// Attempt recovery if enabled
		if browserSession.AutoRecovery {
			log.Printf("🔄 Attempting to recover browser session %s", session.BrowserID)
			if err := GetBrowserManager().recoverSession(session.BrowserID); err != nil {
				log.Printf("❌ Recovery failed for session %s: %v", session.BrowserID, err)
				http.Error(w, "Browser session unhealthy and recovery failed", http.StatusInternalServerError)
				return
			}

			// Get updated browser session after recovery
			browserSession = GetBrowserManager().GetSession(session.BrowserID)
			if browserSession == nil {
				http.Error(w, "Browser session not found after recovery", http.StatusInternalServerError)
				return
			}

			log.Printf("✅ Browser session recovered successfully for %s", session.BrowserID)
		} else {
			http.Error(w, "Browser session unhealthy", http.StatusInternalServerError)
			return
		}
	}

	// Test CDP endpoint accessibility before creating client
	log.Printf("🔍 Testing CDP endpoint accessibility: %s", browserSession.CDPEndpoint)

	// Create persistent CDP connection with retry
	var cdpClient *CDPClient
	var connectErr error
	maxConnectRetries := 3

	for attempt := 1; attempt <= maxConnectRetries; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/%d)", attempt, maxConnectRetries)

		cdpClient = NewCDPClient(browserSession.CDPEndpoint)
		connectErr = cdpClient.Connect()

		if connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}

		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)

		if attempt < maxConnectRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after %d attempts: %v", maxConnectRetries, connectErr)
		http.Error(w, fmt.Sprintf("Failed to connect to browser: %v", connectErr), http.StatusInternalServerError)
		return
	}
	defer cdpClient.Close()

	// Validate connection by getting page info
	_, err := cdpClient.SendCommand("Page.getNavigationHistory", map[string]interface{}{})
	if err != nil {
		log.Printf("⚠️ CDP connection validation failed: %v", err)
		log.Printf("⚠️ This may cause streaming issues, but continuing...")
	} else {
		log.Printf("✅ CDP connection validated for browser %s", session.BrowserID)
	}

	frameCount := 0
	// Adaptive frame rate based on activity
	baseFrameRate := 300 * time.Millisecond // 3.3 FPS base rate
	activeFrameRate := 500 * time.Millisecond // 2 FPS during interactions
	currentFrameRate := baseFrameRate
	
	ticker := time.NewTicker(currentFrameRate)
	defer ticker.Stop()

	// Add timeout for streaming (10 minutes max for automation)
	timeout := time.After(10 * time.Minute)

	// Track when task completes
	completionTime := time.Time{}
	lastSuccessfulFrame := time.Now()
	connectionErrors := 0
	maxConnectionErrors := 8 // Increased tolerance
	consecutiveErrors := 0
	maxConsecutiveErrors := 4 // More forgiving
	lastFrameRateChange := time.Now()

	for {
		select {
		case <-ticker.C:
			// Check if task is completed and track completion time
			GetScriptSessionManager().mutex.RLock()
			currentStatus := session.Status
			GetScriptSessionManager().mutex.RUnlock()

			if currentStatus == "completed" && completionTime.IsZero() {
				completionTime = time.Now()
				log.Printf("📸 Task completed, streaming final result for 10 seconds")
			}

			// Stop streaming when task fails, but wait a bit to ensure it's really failed
			if currentStatus == "failed" {
				// Wait 2 seconds to ensure task is really failed and not just parsing error
				time.Sleep(2 * time.Second)

				// Check status again to confirm it's still failed
				GetScriptSessionManager().mutex.RLock()
				finalStatus := session.Status
				GetScriptSessionManager().mutex.RUnlock()

				if finalStatus == "failed" {
					log.Printf("📸 Stopping stream for session %s (task confirmed failed)", sessionID)
					browserSession.Streaming = false // Reset flag before returning
					return
				} else {
					log.Printf("📸 Task status changed from failed to %s, continuing stream", finalStatus)
				}
			}

			// Continue streaming when task is paused - streaming doesn't stop during pause
			if currentStatus == "paused" {
				log.Printf("📸 Task paused but continuing stream for session %s", sessionID)
				// Don't return - keep streaming during pause
			}

			// Stop streaming after 5 seconds of completion (reduced from 10)
			if currentStatus == "completed" && !completionTime.IsZero() && time.Since(completionTime) > 5*time.Second {
				log.Printf("📸 Stopping stream for session %s (task completed)", sessionID)
				browserSession.Streaming = false // Reset flag before returning
				return
			}

			// Adaptive frame rate based on recent activity
			timeSinceActivity := time.Since(browserSession.LastActivity)
			
			// Switch to slower frame rate during active interactions
			if timeSinceActivity < 3*time.Second {
				if currentFrameRate != activeFrameRate && time.Since(lastFrameRateChange) > 1*time.Second {
					ticker.Stop()
					ticker = time.NewTicker(activeFrameRate)
					currentFrameRate = activeFrameRate
					lastFrameRateChange = time.Now()
					log.Printf("📸 Switched to slower frame rate during interaction (2 FPS)")
				}
				// Longer delay during active interactions
				time.Sleep(1 * time.Second)
			} else if currentFrameRate != baseFrameRate && time.Since(lastFrameRateChange) > 2*time.Second {
				// Switch back to normal frame rate when no recent activity
				ticker.Stop()
				ticker = time.NewTicker(baseFrameRate)
				currentFrameRate = baseFrameRate
				lastFrameRateChange = time.Now()
				log.Printf("📸 Switched back to normal frame rate (3.3 FPS)")
			}

			// Capture screenshot using persistent CDP connection with retry
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				connectionErrors++
				consecutiveErrors++
				log.Printf("⚠️ Failed to capture screenshot after retries (error %d/%d, consecutive %d/%d): %v",
					connectionErrors, maxConnectionErrors, consecutiveErrors, maxConsecutiveErrors, err)

				// Stop if too many consecutive errors (more aggressive)
				if consecutiveErrors >= maxConsecutiveErrors {
					log.Printf("❌ Too many consecutive errors (%d), stopping stream", consecutiveErrors)
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Try to reconnect if connection is persistently lost (after multiple errors)
				if connectionErrors >= 3 && connectionErrors%3 == 0 {
					log.Printf("🔄 Attempting to reconnect CDP after %d errors...", connectionErrors)
					cdpClient.Close()

					// Try to reconnect
					newClient := NewCDPClient(browserSession.CDPEndpoint)
					if reconnectErr := newClient.Connect(); reconnectErr == nil {
						cdpClient = newClient
						log.Printf("✅ CDP reconnected successfully")
						consecutiveErrors = 0 // Reset consecutive errors on successful reconnect
					} else {
						log.Printf("❌ CDP reconnection failed: %v", reconnectErr)
					}
				}

				// Stop if too many total connection errors
				if connectionErrors >= maxConnectionErrors {
					log.Printf("❌ Too many total connection errors (%d), stopping stream", connectionErrors)
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Check if we haven't had a successful frame in too long
				if time.Since(lastSuccessfulFrame) > 15*time.Second {
					log.Printf("❌ No successful frames for 15 seconds, stopping stream")
					browserSession.Streaming = false // Reset flag before returning
					return
				}

				// Skip this frame, keep last image visible
				continue
			} else {
				// Reset error counts on successful capture
				connectionErrors = 0
				consecutiveErrors = 0
				lastSuccessfulFrame = time.Now()

				// Log health status periodically
				if frameCount > 0 && frameCount%50 == 0 {
					log.Printf("📹 Stream healthy: %d frames captured successfully", frameCount)
				}
			}

			// Validate image data - skip if too small
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), likely blank, skipping frame", len(imageData))
				continue
			}

			// Write MJPEG frame
			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(imageData))
			w.Write(imageData)
			fmt.Fprintf(w, "\r\n")

			frameCount++
			if frameCount%10 == 0 {
				log.Printf("📹 Streamed %d frames for session %s (browser: %s)", frameCount, sessionID, session.BrowserID)
			}

			// Small delay to reduce load and improve stability
			time.Sleep(100 * time.Millisecond)

		case <-timeout:
			log.Printf("📸 Streaming timeout for session %s", sessionID)
			browserSession.Streaming = false // Reset flag before returning
			return
		}
	}
}

// HandleWebSocket handles WebSocket connections for real-time communication
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := GetUpgrader().Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WebSocketClient{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		ID:     fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Active: true,
	}

	GetWSManager().register <- client

	go client.WritePump()
	go client.ReadPump()
}

func HandleStreamWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Get session
	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[sessionID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get browser session
	browserSession := GetBrowserManager().GetSession(session.BrowserID)
	if browserSession == nil {
		http.Error(w, "Browser session not found", http.StatusNotFound)
		return
	}

	// Validate browser session is still active
	if browserSession.CDPEndpoint == "" {
		http.Error(w, "Browser session has no CDP endpoint", http.StatusInternalServerError)
		return
	}

	// Upgrade to WebSocket
	conn, err := GetUpgrader().Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Stream WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("📹 WebSocket stream connected for session %s", sessionID)

	// Create CDP client with retry logic
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)

	// Try to connect with retries
	var connectErr error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("🔗 Attempting CDP connection (attempt %d/3)", attempt)
		if connectErr = cdpClient.Connect(); connectErr == nil {
			log.Printf("✅ CDP connection successful on attempt %d", attempt)
			break
		}
		log.Printf("⚠️ CDP connection failed (attempt %d): %v", attempt, connectErr)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if connectErr != nil {
		log.Printf("❌ Failed to connect to CDP after 3 attempts: %v", connectErr)
		// Check if connection is still open before sending error
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Failed to connect to browser"}`)); err != nil {
			log.Printf("⚠️ Failed to send error message to closed WebSocket: %v", err)
		}
		return
	}
	defer cdpClient.Close()

	// Stream frames
	frameCount := 0
	ticker := time.NewTicker(500 * time.Millisecond) // 2 FPS to reduce load and prevent timeouts
	defer ticker.Stop()

	// Mark this session as streaming to prevent conflicts
	browserSession.mutex.Lock()
	if browserSession.Streaming {
		browserSession.mutex.Unlock()
		log.Printf("⚠️ Stream already active for session %s, rejecting new stream request", sessionID)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"Stream already active"}`))
		return
	}
	browserSession.Streaming = true
	browserSession.mutex.Unlock()

	// Ensure streaming flag is reset when done
	defer func() {
		browserSession.mutex.Lock()
		browserSession.Streaming = false
		browserSession.mutex.Unlock()
		log.Printf("📸 Stream ended for session %s", sessionID)
	}()

	// Set up ping/pong to detect disconnections
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Channel to handle graceful shutdown
	done := make(chan struct{})

	// Start goroutine to handle ping/pong and detect disconnections
	go func() {
		defer close(done)
		for {
			select {
			case <-pingTicker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("📸 WebSocket ping failed, client disconnected: %v", err)
					return
				}
			case <-r.Context().Done():
				log.Printf("📸 WebSocket stream client disconnected for session %s", sessionID)
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			// Check if task is completed
			GetScriptSessionManager().mutex.RLock()
			currentStatus := session.Status
			GetScriptSessionManager().mutex.RUnlock()

			// Stop streaming when task completes or fails
			if currentStatus == "completed" || currentStatus == "failed" {
				log.Printf("📸 Stopping WebSocket stream for session %s (task %s)", sessionID, currentStatus)
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"task_complete","status":"`+currentStatus+`"}`)); err != nil {
					log.Printf("⚠️ Failed to send task complete message: %v", err)
				}
				return
			}

			// Continue streaming when task is paused - just send a status message
			if currentStatus == "paused" {
				log.Printf("📸 Task paused but continuing WebSocket stream for session %s", sessionID)
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"task_status","status":"paused","message":"Task paused but streaming continues"}`)); err != nil {
					log.Printf("⚠️ Failed to send task paused message: %v", err)
				}
				// Continue streaming instead of returning
			}

			// Capture screenshot with timeout and retry
			imageData, err := cdpClient.CaptureScreenshotWithRetry(3)
			if err != nil {
				log.Printf("⚠️ Failed to capture screenshot after retries: %v", err)
				// Try to reconnect CDP client
				if reconnectErr := cdpClient.Reconnect(); reconnectErr != nil {
					log.Printf("❌ Failed to reconnect CDP client: %v", reconnectErr)
					conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"CDP connection lost"}`))
					return
				}
				log.Printf("✅ CDP client reconnected successfully")
				continue
			}

			// Validate image data
			if len(imageData) < 1000 {
				log.Printf("⚠️ Screenshot too small (%d bytes), skipping frame", len(imageData))
				continue
			}

			// Send frame via WebSocket
			frameCount++
			if frameCount%25 == 0 { // Reduced logging frequency
				log.Printf("📹 WebSocket stream healthy: %d frames sent", frameCount)
			}

			// Convert to base64 and send
			frameData := map[string]interface{}{
				"type": "frame",
				"data": base64.StdEncoding.EncodeToString(imageData),
			}

			// Set write deadline to prevent hanging
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteJSON(frameData); err != nil {
				log.Printf("❌ Failed to send frame via WebSocket: %v", err)
				return
			}

		case <-done:
			log.Printf("📸 WebSocket stream done for session %s", sessionID)
			return
		case <-r.Context().Done():
			log.Printf("📸 Client disconnected for session %s", sessionID)
			return
		}
	}
}

// GetTaskResultHandler returns detailed result information for a task
func GetTaskResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[taskID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	// Generate base URL for links
	baseURL := generateBaseURL(r)
	
	// Build the detailed result response
	result := &TaskResultResponse{
		ID:                taskID,
		Task:              session.Task,
		LiveURL:           fmt.Sprintf("%s/api/live-automation/%s", baseURL, taskID),
		Output:            session.Output,
		Status:            session.Status,
		CreatedAt:         session.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		Steps:             session.Steps,
		BrowserData:       BrowserData{Cookies: []interface{}{}},
		UserUploadedFiles: []string{},
		OutputFiles:       []string{},
		PublicShareURL:    fmt.Sprintf("%s/session/%s/result", baseURL, taskID),
		Summary:           session.Summary,
		Metadata: TaskMetadata{
			SessionID:     session.BrowserID,
			Logs:          session.Logs,
			LogsSummary:   calculateLogsSummary(session.Logs),
		},
		TokenUsage:        session.TokenUsage, // Include token usage data
	}

	// Set FinishedAt if the task is completed
	if session.FinishedAt != nil {
		finishedAtStr := session.FinishedAt.Format("2006-01-02T15:04:05.000Z")
		result.FinishedAt = &finishedAtStr
	}

	// Calculate duration
	var endTime time.Time
	if session.FinishedAt != nil {
		endTime = *session.FinishedAt
	} else {
		endTime = time.Now()
	}
	duration := endTime.Sub(session.CreatedAt).Milliseconds()
	result.Metadata.Duration = duration
	result.Metadata.DurationHuman = formatDuration(duration)

	// Set default output if empty
	if result.Output == "" {
		if session.Status == "completed" {
			result.Output = fmt.Sprintf("Task executed successfully! Browser is now available at: %s", result.LiveURL)
		} else if session.Status == "failed" {
			result.Output = "Task execution failed"
		} else if session.Status == "running" {
			result.Output = "Task is currently running"
		} else {
			result.Output = "Task is queued for execution"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// GetTaskStatusHandler returns simple status for a task
func GetTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[taskID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	response := &TaskStatusResponse{
		Status: session.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetTaskExecutionStatusHandler returns task execution status for Python script polling
func GetTaskExecutionStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	GetScriptSessionManager().mutex.RLock()
	session := GetScriptSessionManager().sessions[taskID]
	GetScriptSessionManager().mutex.RUnlock()

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Task not found",
			"should_continue": false,
		})
		return
	}

	// Determine if the agent should continue execution
	shouldContinue := session.Status == "running"
	isPaused := session.Status == "paused"
	isStopped := session.Status == "stopped" || session.Status == "cancelled" || session.Status == "failed"

	response := map[string]interface{}{
		"status":          session.Status,
		"should_continue": shouldContinue,
		"is_paused":       isPaused,
		"is_stopped":      isStopped,
		"task_id":         taskID,
		"timestamp":       time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StopTaskHandler stops a running task while keeping the browser session alive
func StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	log.Printf("🛑 Stop request received for task ID: %s", taskID)

	GetScriptSessionManager().mutex.Lock()
	session := GetScriptSessionManager().sessions[taskID]
	if session == nil {
		GetScriptSessionManager().mutex.Unlock()
		log.Printf("❌ Task not found: %s", taskID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	log.Printf("📊 Current task status: %s", session.Status)

	// Stop the task but keep browser session alive
	if session.Status == "running" || session.Status == "pending" {
		session.Status = "stopped"
		session.UpdatedAt = time.Now()
		now := time.Now()
		session.FinishedAt = &now
		
		// If there's a process running, kill it
		if session.Process != nil {
			log.Printf("🔪 Killing process with PID: %d", session.Process.Process.Pid)
			// Check if process is still running
			if session.Process.ProcessState == nil || !session.Process.ProcessState.Exited() {
				err := session.Process.Process.Kill()
				if err != nil {
					log.Printf("⚠️ Error killing process: %v", err)
				} else {
					log.Printf("✅ Process killed successfully")
				}
			} else {
				log.Printf("ℹ️ Process already exited")
			}
			// Clear the process reference
			session.Process = nil
		} else {
			log.Printf("⚠️ No process found to kill (task may have already completed)")
		}
		
		log.Printf("✅ Task stopped successfully: %s", taskID)
	} else {
		log.Printf("ℹ️ Task %s is already in status: %s", taskID, session.Status)
	}
	GetScriptSessionManager().mutex.Unlock()

	// Also reset the browser session to "ready" state so it can be reused
	if session.BrowserID != "" {
		browserSession := GetBrowserManager().GetSession(session.BrowserID)
		if browserSession != nil {
			browserSession.mutex.Lock()
			browserSession.Status = "ready"
			browserSession.TaskCompleted = false  // Reset task completed flag
			browserSession.LastActivity = time.Now()
			browserSession.mutex.Unlock()
			
			// Cancel cleanup timer to prevent auto-deletion
			GetBrowserManager().cancelCleanupTimer(session.BrowserID)
			log.Printf("✅ Browser session %s reset to ready state for reuse", session.BrowserID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "stopped",
		"message": "Task stopped successfully. Browser session remains active.",
		"task_id": taskID,
	})
}

// PauseTaskHandler pauses a running task
func PauseTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	log.Printf("⏸️ Pause request received for task ID: %s", taskID)

	GetScriptSessionManager().mutex.Lock()
	session := GetScriptSessionManager().sessions[taskID]
	if session == nil {
		GetScriptSessionManager().mutex.Unlock()
		log.Printf("❌ Task not found: %s", taskID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	log.Printf("📊 Current task status: %s", session.Status)

	// Only allow pausing if the task is currently running
	if session.Status != "running" {
		GetScriptSessionManager().mutex.Unlock()
		log.Printf("⚠️ Cannot pause task %s with status: %s", taskID, session.Status)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Cannot pause task with status '%s'. Task must be running to pause.", session.Status),
			"current_status": session.Status,
		})
		return
	}

	// Pause the task by setting status - don't kill the process
	session.Status = "paused"
	session.UpdatedAt = time.Now()
	
	// Store pause metadata - but don't kill the process
	session.Metadata["paused_at"] = time.Now().Format(time.RFC3339)
	session.Metadata["pause_reason"] = "user_requested"
	
	log.Printf("⏸️ Task paused (status only), Python script will handle pause logic")
	if session.Process != nil {
		log.Printf("ℹ️ Process PID %d will continue running but agent execution will pause", session.Process.Process.Pid)
	}

	GetScriptSessionManager().mutex.Unlock()

	log.Printf("✅ Task paused successfully: %s", taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "paused",
		"message": "Task paused successfully. Use resume endpoint to continue.",
		"task_id": taskID,
		"paused_at": time.Now().Format(time.RFC3339),
	})
}

// ResumeTaskHandler resumes a paused task
func ResumeTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	log.Printf("▶️ Resume request received for task ID: %s", taskID)

	GetScriptSessionManager().mutex.Lock()
	session := GetScriptSessionManager().sessions[taskID]
	if session == nil {
		GetScriptSessionManager().mutex.Unlock()
		log.Printf("❌ Task not found: %s", taskID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	log.Printf("📊 Current task status: %s", session.Status)

	// Only allow resuming if the task is currently paused
	if session.Status != "paused" {
		GetScriptSessionManager().mutex.Unlock()
		log.Printf("⚠️ Cannot resume task %s with status: %s", taskID, session.Status)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Cannot resume task with status '%s'. Task must be paused to resume.", session.Status),
			"current_status": session.Status,
		})
		return
	}

	// Resume the task by changing status - Python script will detect this
	session.Status = "running"
	session.UpdatedAt = time.Now()
	
	// Clear pause metadata and add resume info
	if pausedAt, exists := session.Metadata["paused_at"]; exists {
		if pausedAtStr, ok := pausedAt.(string); ok {
			session.Metadata["resumed_at"] = time.Now().Format(time.RFC3339)
			session.Metadata["pause_duration"] = calculatePauseDuration(pausedAtStr)
		}
		delete(session.Metadata, "paused_at")
		delete(session.Metadata, "pause_reason")
	}
	
	log.Printf("▶️ Task resumed (status changed), Python script will detect and continue execution")
	if session.Process != nil {
		log.Printf("ℹ️ Process PID %d will continue with agent execution resuming", session.Process.Process.Pid)
	}

	GetScriptSessionManager().mutex.Unlock()

	log.Printf("✅ Task resumed successfully: %s", taskID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "running",
		"message": "Task resumed successfully. Execution will continue.",
		"task_id": taskID,
		"resumed_at": time.Now().Format(time.RFC3339),
	})
}

// Helper function to calculate pause duration
func calculatePauseDuration(pausedAtStr string) string {
	pausedAt, err := time.Parse(time.RFC3339, pausedAtStr)
	if err != nil {
		return "unknown"
	}
	
	duration := time.Since(pausedAt)
	if duration < time.Minute {
		return fmt.Sprintf("%d seconds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%d minutes", int(duration.Minutes()))
	} else {
		return fmt.Sprintf("%.1f hours", duration.Hours())
	}
}
