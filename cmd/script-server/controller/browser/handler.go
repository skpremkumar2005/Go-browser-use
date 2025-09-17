package browser

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"go-webrtc/cmd/script-server/config"
	"go-webrtc/cmd/script-server/libs/shared/helpers"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// Handler interface defines HTTP handlers for browser operations
type Handler interface {
	Route(g *echo.Group)

	// Browser Session Handlers
	CreateBrowserSession(c echo.Context) error
	GetBrowserSession(c echo.Context) error
	ListBrowserSessions(c echo.Context) error

	// Script Task Handlers
	CreateScriptTask(c echo.Context) error
	GetScriptTask(c echo.Context) error
	ListScriptTasks(c echo.Context) error

	// Browser Action Handlers
	ExecuteBrowserAction(c echo.Context) error
	CaptureScreenshot(c echo.Context) error

	// System Status Handlers
	GetSystemStatus(c echo.Context) error

	// Streaming Handlers
	HandleWebSocket(c echo.Context) error
	StartStreamingSession(c echo.Context) error
	StopStreamingSession(c echo.Context) error
	ResetStreamingSession(c echo.Context) error
	StreamScreencast(c echo.Context) error
	
	// Execute endpoint (same as old code)
	ExecuteTaskAndStream(c echo.Context) error

	// Live Page Handlers
	LiveAutomationPage(c echo.Context) error
}

// handlerFixed implements the Handler interface with fixed streaming
type handler struct {
	service Service
	config  *config.Config
}

// NewHandlerFixed creates a new handler instance with fixes
func NewHandler(service Service, cfg *config.Config) Handler {
       return &handler{
	       service: service,
	       config:  cfg,
       }
}

// Browser Session Handlers (same as original)
func (h *handler) CreateBrowserSession(c echo.Context) error {
	data := c.Get("createBrowserSession").(CreateBrowserSessionDto)
	
	session, err := h.service.CreateBrowserSession(data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Browser session created successfully", session)
}

func (h *handler) GetBrowserSession(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	session, err := h.service.GetBrowserSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Browser session retrieved successfully", session)
}

func (h *handler) ListBrowserSessions(c echo.Context) error {
	sessions, err := h.service.ListBrowserSessions()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Browser sessions retrieved successfully", map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// Script Task Handlers (same as original)
func (h *handler) CreateScriptTask(c echo.Context) error {
	data := c.Get("scriptTaskRequest").(ScriptTaskRequestDto)
	
	task, err := h.service.CreateScriptTask(data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Script task created successfully", task)
}

func (h *handler) GetScriptTask(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	task, err := h.service.GetScriptTask(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Script task retrieved successfully", task)
}

func (h *handler) ListScriptTasks(c echo.Context) error {
	tasks, err := h.service.ListScriptTasks()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Script tasks retrieved successfully", map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// Browser Action Handlers (same as original)
func (h *handler) ExecuteBrowserAction(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	action := c.Get("browserAction").(BrowserActionDto)
	actionData := c.Get("actionData")
	
	err := h.service.ExecuteBrowserAction(sessionID, action, actionData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Browser action executed successfully", nil)
}

func (h *handler) CaptureScreenshot(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	screenshot, err := h.service.CaptureScreenshot(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Screenshot captured successfully", map[string]string{
		"image": screenshot,
	})
}

// System Status Handlers (same as original)
func (h *handler) GetSystemStatus(c echo.Context) error {
	status, err := h.service.GetSystemStatus()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "System status retrieved successfully", status)
}

// HandleWebSocket handles WebSocket connections with streaming capability
func (h *handler) HandleWebSocket(c echo.Context) error {
	sessionID := c.Param("sessionId")

	if sessionID == "" {
		log.Printf("❌ No session ID provided")
		return c.String(http.StatusBadRequest, "Session ID is required")
	}

	log.Printf("🔌 Attempting WebSocket upgrade for session %s", sessionID)

	// Check if the response writer supports hijacking
	if _, ok := c.Response().Writer.(http.Hijacker); !ok {
		log.Printf("❌ Response writer does not support hijacking - middleware interference detected")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "WebSocket upgrade not supported - middleware configuration issue",
			"code": "WEBSOCKET_NOT_SUPPORTED",
		})
	}

	// Upgrade to WebSocket connection with error handling
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		// Don't return error here as the connection might already be upgraded
		if c.Response().Committed {
			log.Printf("⚠️ Response already committed, WebSocket upgrade may have partially succeeded")
			return nil
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "WebSocket upgrade failed",
			"details": err.Error(),
		})
	}
	defer ws.Close()

	log.Printf("✅ WebSocket connected for session %s", sessionID)
	
	// Get browser session
	browserSession, err := h.service.GetBrowserSession(sessionID)
	if err != nil {
		log.Printf("❌ Session %s not found: %v", sessionID, err)
		ws.WriteJSON(map[string]interface{}{
			"type": "error",
			"message": "Session not found",
			"code": "SESSION_NOT_FOUND",
		})
		return nil
	}

	if browserSession.Status == "closed" {
		log.Printf("⚠️ Browser session %s is closed", sessionID)
		ws.WriteJSON(map[string]interface{}{
			"type": "error",
			"message": "Browser session is closed",
			"code": "SESSION_CLOSED",
		})
		return nil
	}

	// Check if CDP endpoint is available
	if browserSession.CDPEndpoint == "" {
		log.Printf("⚠️ CDP endpoint not available for session %s", sessionID)
		ws.WriteJSON(map[string]interface{}{
			"type": "error",
			"message": "CDP endpoint not available",
			"code": "CDP_NOT_READY",
		})
		return nil
	}

	log.Printf("🔗 Using CDP endpoint: %s", browserSession.CDPEndpoint)

	// Create CDP client for screenshot capture
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		log.Printf("❌ Failed to connect to CDP: %v", err)
		ws.WriteJSON(map[string]interface{}{
			"type": "error",
			"message": "Failed to connect to browser",
			"code": "CDP_CONNECTION_FAILED",
		})
		return nil
	}
	defer cdpClient.Close()

	// Send connection success message
	ws.WriteJSON(map[string]interface{}{
		"type": "connected",
		"sessionId": sessionID,
		"message": "WebSocket streaming started",
		"timestamp": time.Now().Unix(),
	})

	// Start streaming loop
	ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
	defer ticker.Stop()

	frameCount := 0
	maxFrames := 12000 // 10 minutes at 20 FPS
	connectionErrors := 0
	maxErrors := 10
	lastValidFrame := []byte{}

	log.Printf("📹 WebSocket streaming started for session %s", sessionID)

	// Get request context for cancellation
	ctx := c.Request().Context()

	// Main streaming loop
	for {
		select {
		case <-ctx.Done():
			log.Printf("📸 Client disconnected from WebSocket stream %s after %d frames", sessionID, frameCount)
			return nil

		case <-ticker.C:
			// Check limits
			if frameCount >= maxFrames {
				log.Printf("📹 Max frames reached (%d), stopping stream", maxFrames)
				ws.WriteJSON(map[string]interface{}{
					"type": "stream_ended",
					"reason": "max_frames_reached",
					"frames_sent": frameCount,
				})
				return nil
			}

			if connectionErrors >= maxErrors {
				log.Printf("❌ Too many connection errors (%d), stopping stream", connectionErrors)
				ws.WriteJSON(map[string]interface{}{
					"type": "stream_ended",
					"reason": "too_many_errors",
					"frames_sent": frameCount,
				})
				return nil
			}

			// Check if session is still active
			currentSession, err := h.service.GetBrowserSession(sessionID)
			if err != nil {
				log.Printf("❌ Failed to get session status: %v", err)
				connectionErrors++
				continue
			}

			if currentSession.Status == "closed" {
				log.Printf("📸 Browser session %s closed, stopping stream", currentSession.Status)
				ws.WriteJSON(map[string]interface{}{
					"type": "stream_ended",
					"reason": "session_closed",
					"frames_sent": frameCount,
				})
				return nil
			}

			// Capture screenshot
			imageData, err := cdpClient.CaptureScreenshot()
			if err != nil {
				connectionErrors++
				log.Printf("⚠️ Screenshot failed (error %d/%d): %v", connectionErrors, maxErrors, err)
				
				// Use cached frame if available
				if len(lastValidFrame) > 0 {
					log.Printf("🔄 Using cached frame due to screenshot error")
					imageData = lastValidFrame
					
					// Send cached frame
					ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
					if err := ws.WriteMessage(websocket.BinaryMessage, imageData); err != nil {
						log.Printf("❌ WebSocket write error: %v", err)
						return nil
					}
					connectionErrors = 0 // Reset errors since we sent a frame
					frameCount++
					continue
				}
				
				// Wait longer for CDP-specific errors
				if strings.Contains(err.Error(), "CDP endpoint not yet available") {
					time.Sleep(2 * time.Second)
				}
				continue
			}

			// Validate image data
			if len(imageData) < 1000 {
				connectionErrors++
				log.Printf("⚠️ Screenshot too small (%d bytes), skipping frame", len(imageData))
				
				if len(lastValidFrame) > 0 {
					imageData = lastValidFrame
				} else {
					continue
				}
			}

			// Cache valid frame
			lastValidFrame = make([]byte, len(imageData))
			copy(lastValidFrame, imageData)
			connectionErrors = 0

			frameCount++

			// Send frame via WebSocket
			ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
			if err := ws.WriteMessage(websocket.BinaryMessage, imageData); err != nil {
				log.Printf("❌ WebSocket write error: %v", err)
				return nil
			}

			// Log progress
			if frameCount%400 == 0 {
				log.Printf("📹 Streamed %d frames via WebSocket for session %s", frameCount, sessionID)
			}

		default:
			// Handle incoming WebSocket messages (non-blocking)
			ws.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			messageType, message, err := ws.ReadMessage()
			
			if err != nil {
				// Check if it's just a timeout (expected for non-blocking read)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Continue streaming loop
				}
				// Real error - client disconnected
				log.Printf("❌ WebSocket read error: %v", err)
				return nil
			}

			// Handle different message types
			switch messageType {
			case websocket.TextMessage:
				// Handle text messages (commands, etc.)
				log.Printf("📨 WebSocket text message: %s", string(message))
				
				// Echo response for now
				response := map[string]interface{}{
					"type": "response",
					"message": "Message received",
					"original": string(message),
					"timestamp": time.Now().Unix(),
				}
				
				ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
				if err := ws.WriteJSON(response); err != nil {
					log.Printf("❌ WebSocket write error: %v", err)
					return nil
				}

			case websocket.BinaryMessage:
				// Handle binary messages if needed
				log.Printf("� WebSocket binary message received (%d bytes)", len(message))

			case websocket.CloseMessage:
				log.Printf("🔌 WebSocket close message received")
				return nil
			}
		}
	}
}

func (h *handler) StartStreamingSession(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	err := h.service.StartStreamingSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Streaming session started successfully", nil)
}

func (h *handler) StopStreamingSession(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	err := h.service.StopStreamingSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Streaming session stopped successfully", nil)
}

func (h *handler) ResetStreamingSession(c echo.Context) error {
	sessionID := c.Get("sessionId").(string)
	
	err := h.service.ResetStreamingSession(sessionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	return helpers.RespSuccess(c, "Streaming session reset successfully", nil)
}

// FIXED StreamScreencast - properly uses service methods instead of type casting
func (h *handler) StreamScreencast(c echo.Context) error {
	sessionID := c.Param("sessionId")
	
	if sessionID == "" {
		log.Printf("❌ No session ID provided")
		return c.String(http.StatusBadRequest, "Session ID is required")
	}

	log.Printf("📹 Starting MJPEG stream for session: %s", sessionID)

	// Get browser session via service (correct approach)
	browserSession, err := h.service.GetBrowserSession(sessionID)
	if err != nil {
		log.Printf("❌ Session %s not found: %v", sessionID, err)
		// Return JSON error for frontend to handle
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Session not found. Please create a new session first.",
			"code": "SESSION_NOT_FOUND",
			"sessionId": sessionID,
		})
	}

	if browserSession.Status == "closed" {
		log.Printf("⚠️ Browser session %s is closed, cannot stream", sessionID)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Browser session is closed",
			"code": "SESSION_CLOSED",
			"sessionId": sessionID,
		})
	}

	// Check if CDP endpoint is available
	if browserSession.CDPEndpoint == "" {
		log.Printf("⚠️ CDP endpoint not available for session %s", sessionID)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"error": "CDP endpoint not available - browser may still be starting",
			"code": "CDP_NOT_READY",
			"sessionId": sessionID,
		})
	}

	log.Printf("🔗 Using CDP endpoint: %s", browserSession.CDPEndpoint)

	// Create persistent CDP client for streaming
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		log.Printf("❌ Failed to connect to CDP: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"error": "Failed to connect to browser CDP endpoint",
			"code": "CDP_CONNECTION_FAILED",
			"sessionId": sessionID,
		})
	}
	defer cdpClient.Close()

	// Set MJPEG headers (same as working old code)
	c.Response().Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Expires", "0")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	w := c.Response().Writer
	c.Response().WriteHeader(http.StatusOK)
	
	// Try to get flusher interface
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
		log.Printf("✅ Streaming supported, flusher interface available")
	} else {
		log.Printf("⚠️ No flusher interface, will try streaming anyway")
	}

	// Streaming loop - Increased frame rate for live streaming
	ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS for smooth live streaming
	defer ticker.Stop()

	frameCount := 0
	maxFrames := 12000 // 10 minutes at 20 FPS
	connectionErrors := 0
	maxErrors := 10 // Increased error tolerance for higher frame rate
	lastValidFrame := []byte{} // Cache last valid frame

	// Write initial boundary
	if _, err := w.Write([]byte("--frame\r\n")); err != nil {
		log.Printf("❌ Failed to write initial boundary: %v", err)
		return nil
	}

	log.Printf("📹 MJPEG streaming started for session %s (no timeout)", sessionID)

	ctx := c.Request().Context()

	// Streaming loop with proper error handling
	for {
		select {
		case <-ctx.Done():
			log.Printf("📸 Client disconnected from stream %s after %d frames", sessionID, frameCount)
			return nil

		case <-ticker.C:
			// Check if we've exceeded limits
			if frameCount >= maxFrames {
				log.Printf("📹 Max frames reached (%d), stopping stream", maxFrames)
				return nil
			}

			if connectionErrors >= maxErrors {
				log.Printf("❌ Too many connection errors (%d), stopping stream", connectionErrors)
				return nil
			}

			// Get fresh browser session status
			currentSession, err := h.service.GetBrowserSession(sessionID)
			if err != nil {
				log.Printf("❌ Failed to get session status: %v", err)
				connectionErrors++
				continue
			}

			// Stop streaming when browser session is closed
			if currentSession.Status == "closed" {
				log.Printf("📸 Browser session %s, stopping stream", currentSession.Status)
				return nil
			}

			// Capture screenshot directly via persistent CDP client (much faster)
			imageData, err := cdpClient.CaptureScreenshot()
			if err != nil {
				connectionErrors++
				log.Printf("⚠️ Screenshot failed (error %d/%d): %v", connectionErrors, maxErrors, err)
				
				// If we have a cached frame, use it instead of failing
				if len(lastValidFrame) > 0 {
					log.Printf("🔄 Using cached frame due to screenshot error")
					// Use cached frame
					imageData := lastValidFrame
					
					// Write MJPEG frame with cached data
					if _, err := w.Write([]byte("Content-Type: image/jpeg\r\n")); err != nil {
						log.Printf("❌ Client disconnected during cached frame header")
						return nil
					}
					if _, err := w.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(imageData)))); err != nil {
						log.Printf("❌ Client disconnected during cached frame length")
						return nil
					}
					if _, err := w.Write(imageData); err != nil {
						log.Printf("❌ Client disconnected during cached frame data")
						return nil
					}
					if _, err := w.Write([]byte("\r\n--frame\r\n")); err != nil {
						log.Printf("❌ Client disconnected during cached frame boundary")
						return nil
					}
					
					// Flush and continue without counting as error
					if flusher != nil {
						flusher.Flush()
					}
					connectionErrors = 0 // Reset errors since we successfully sent a frame
					frameCount++
					continue
				}
				
				// For CDP-specific errors, wait longer before retrying
				if strings.Contains(err.Error(), "CDP endpoint not yet available") {
					time.Sleep(2 * time.Second)
				}
				continue
			}

			// Validate image data
			if len(imageData) < 1000 {
				connectionErrors++
				log.Printf("⚠️ Screenshot too small (%d bytes), skipping frame", len(imageData))
				
				// Try to use cached frame instead
				if len(lastValidFrame) > 0 {
					imageData = lastValidFrame
				} else {
					continue
				}
			}

			// Cache this valid frame for future use
			lastValidFrame = make([]byte, len(imageData))
			copy(lastValidFrame, imageData)

			// Reset connection errors on successful screenshot
			connectionErrors = 0

			frameCount++

			// Write MJPEG frame (EXACT same format as working old code)
			if _, err := w.Write([]byte("Content-Type: image/jpeg\r\n")); err != nil {
				log.Printf("❌ Client disconnected during frame write")
				return nil
			}

			if _, err := w.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(imageData)))); err != nil {
				log.Printf("❌ Client disconnected during header write")
				return nil
			}

			if _, err := w.Write(imageData); err != nil {
				log.Printf("❌ Client disconnected during data write")
				return nil
			}

			if _, err := w.Write([]byte("\r\n--frame\r\n")); err != nil {
				log.Printf("❌ Client disconnected during boundary write")
				return nil
			}

			// Log progress every 400 frames (20 seconds at 20 FPS)
			if frameCount%400 == 0 {
				log.Printf("📹 Streamed %d frames for session %s", frameCount, sessionID)
			}

			// Flush immediately for real-time streaming
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

// LiveAutomationPage serves the live automation page using live.html template (same as original)
func (h *handler) LiveAutomationPage(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.String(http.StatusBadRequest, "Session ID is required")
	}
	
	// Template data matching the old code structure
	// Get base URL for WebSocket connection
	baseURL := c.Request().Host
	
	data := struct {
		TaskID       string
		Task         string
		Status       string
		CreatedAt    string
		BrowserID    string
		IsAutomation bool
		WebSocketURL string
	}{
		TaskID:       sessionID,
		Task:         "Browser automation session",
		Status:       "active",
		CreatedAt:    time.Now().Format("2006-01-02 15:04:05"),
		BrowserID:    sessionID,
		IsAutomation: true,
		WebSocketURL: fmt.Sprintf("ws://%s/api/v1/browser_use/browser/websocket-stream/%s", baseURL, sessionID),
	}
	
	// Find and parse the live_websocket.html template  
	templatePaths := []string{
		"../../frontend/live_websocket.html",   // From cmd/script-server to root frontend
		"../../../frontend/live_websocket.html", // Alternative path
		"frontend/live_websocket.html",         // Current directory
		"./live_websocket.html",               // Fallback
		"../../frontend/live_fixed.html",      // Fallback to old template
	}
	
	var templatePath string
	for _, path := range templatePaths {
		if _, err := os.Stat(path); err == nil {
			templatePath = path
			break
		}
	}
	
	if templatePath == "" {
		// Fallback: serve a basic HTML page if template not found
		return c.HTML(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head><title>Live Automation</title></head>
		<body>
			<h1>Live Browser Automation</h1>
			<p>Session: `+sessionID+`</p>
			<p>Template not found at expected paths</p>
		</body>
		</html>`)
	}
	
	// Parse and execute the template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("❌ Failed to parse template: %v", err)
		// Fallback to basic HTML
		return c.String(http.StatusInternalServerError, "Template parsing error")
	}
	
	// Set headers to allow inline content (fix CSP issues)
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response().Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob: ws: wss:; img-src 'self' data: blob: http: https:; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss: http: https:;")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	
	// Execute template with data
	return tmpl.Execute(c.Response().Writer, data)
}

// ExecuteTaskAndStream creates browser first, then task and returns response immediately (FIXED)
func (h *handler) ExecuteTaskAndStream(c echo.Context) error {
	// Parse request body with new structure
	var req ExecuteTaskRequestDto
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format: " + err.Error()})
	}
	
	// Validate request
	if req.Task == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task is required"})
	}
	
	if req.MaxSteps == 0 {
		req.MaxSteps = 25 // Default max steps to match request example
	}
	
	// Validate LLM model configuration
	if req.LLMModel.ApiKey == "" || req.LLMModel.Provider == "" || 
	   req.LLMModel.Endpoint == "" || req.LLMModel.Deployment == "" || req.LLMModel.LLMModel == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "All llm_model fields are required (apiKey, provider, endpoint, deployment, llmModel)"})
	}
	
	log.Printf("ExecuteTaskAndStream: %s (max steps: %d, provider: %s)", req.Task, req.MaxSteps, req.LLMModel.Provider)
	
	// STEP 1: Create browser session FIRST (like old code)
	log.Printf("Creating browser session first (old code architecture)")
	
	browserSession, err := h.service.CreateBrowserSession(CreateBrowserSessionDto{
		Viewport: ViewportDto{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create browser session: " + err.Error()})
	}
	
	// Use the returned ID from browser session
	sessionID := browserSession.ID
	log.Printf("Created browser session %s with CDP: %s", sessionID, browserSession.CDPEndpoint)
	
	// Add a small delay to ensure browser is fully ready before connecting Python
	log.Printf("⏳ Allowing browser to stabilize before task execution...")
	time.Sleep(3 * time.Second)
	
	// STEP 2: Create task with existing browser and LLM configuration
	log.Printf("🎯 Creating task with existing browser CDP endpoint and LLM config")
	taskID, err := h.service.CreateScriptTaskWithLLM(ScriptTaskWithLLMDto{
		Task:        req.Task,
		MaxSteps:    req.MaxSteps,
		BrowserID:   browserSession.ID,
		CDPEndpoint: browserSession.CDPEndpoint,
		LLMConfig:   req.LLMModel,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create task: " + err.Error()})
	}
	
	log.Printf("✅ Created task %s for session %s", taskID, sessionID)
	
	// STEP 3: Return response in the exact format requested
	baseURL := fmt.Sprintf("http://%s", c.Request().Host)
	
	// Build live_url and WebSocket URL (converted from MJPEG to WebSocket streaming)
	liveURL := fmt.Sprintf("%s/api/v1/browser_use/browser/live-automation/%s", baseURL, sessionID)
	socketURL := fmt.Sprintf("ws://%s/api/v1/browser_use/browser/websocket-stream/%s", c.Request().Host, sessionID)
	
	response := ExecuteTaskResponseDto{
		Success:       true,
		ID:            taskID,
		SessionID:     sessionID,
		SessionReused: false, // Always false for new sessions
		LiveURL:       liveURL,
		SocketURL:     socketURL,
	}
	
	log.Printf("📤 Returning response with task ID: %s, session ID: %s", taskID, sessionID)
	
	return c.JSON(http.StatusOK, response)
}

// RecoverSession creates a new browser session for recovery purposes
// This helps when the frontend tries to access an expired/lost session
func (h *handler) RecoverSession(c echo.Context) error {
	sessionID := c.Param("sessionId")
	
	// Check if session already exists
	if _, err := h.service.GetBrowserSession(sessionID); err == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Session already exists",
			"sessionId": sessionID,
		})
	}
	
	log.Printf("🔄 Creating recovery browser session for %s", sessionID)
	
	// Create a new browser session with the requested ID
	browserSession, err := h.service.CreateBrowserSession(CreateBrowserSessionDto{
		Viewport: ViewportDto{
			Width:  1920,
			Height: 1080,
		},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	// Create response with all URLs
	baseURL := fmt.Sprintf("http://%s", c.Request().Host)
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":        browserSession.ID,
			"sessionId": browserSession.ID,
			"status":    "recovered",
			"message":   "Recovery session created successfully",
			"createdAt": time.Now().Format(time.RFC3339),
			// URLs for different access methods
			"live_url":      fmt.Sprintf("%s/api/v1/browser_use/browser/live-automation/%s", baseURL, browserSession.ID),
			"streaming_url": fmt.Sprintf("%s/api/v1/browser_use/browser/stream-screencast/%s", baseURL, browserSession.ID),
			"websocket_url": fmt.Sprintf("ws://%s/api/v1/browser_use/browser/ws/%s", c.Request().Host, browserSession.ID),
			"browser_info": map[string]interface{}{
				"browserId":   browserSession.ID,
				"cdpEndpoint": browserSession.CDPEndpoint,
				"viewport":    browserSession.Viewport,
			},
		},
		"message": "Recovery session created and ready for streaming",
	}
	
	return c.JSON(http.StatusOK, response)
}