package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// EnhancedWebSocketHandler provides a robust WebSocket handler with enhanced CDP service
type EnhancedWebSocketHandler struct {
	upgrader websocket.Upgrader
	sessions map[string]*EnhancedSessionManager
	mu       sync.RWMutex
}

// EnhancedSessionManager manages a WebSocket session with enhanced CDP service
type EnhancedSessionManager struct {
	sessionID     string
	wsConn        *websocket.Conn
	cdpService    *EnhancedService
	wsConnMutex   sync.RWMutex
	messageQueue  chan []byte
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Session state
	isActive      bool
	lastActivity  time.Time
	errorCount    int
	maxErrors     int
}

// NewEnhancedWebSocketHandler creates a new enhanced WebSocket handler
func NewEnhancedWebSocketHandler() *EnhancedWebSocketHandler {
	return &EnhancedWebSocketHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
			ReadBufferSize:  32768, // 32KB read buffer
			WriteBufferSize: 32768, // 32KB write buffer
		},
		sessions: make(map[string]*EnhancedSessionManager),
	}
}

// HandleWebSocketEnhanced handles WebSocket connections with enhanced CDP service
func (h *EnhancedWebSocketHandler) HandleWebSocketEnhanced(c echo.Context) error {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Missing session ID",
			"message": "Session ID is required for WebSocket connection",
		})
	}

	log.Printf("🔌 Enhanced WebSocket connection request for session %s", sessionID)

	// Upgrade HTTP connection to WebSocket
	wsConn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return err
	}

	// Create session manager
	ctx, cancel := context.WithCancel(context.Background())
	session := &EnhancedSessionManager{
		sessionID:    sessionID,
		wsConn:       wsConn,
		cdpService:   NewEnhancedService(),
		messageQueue: make(chan []byte, 1000), // Large message buffer
		ctx:          ctx,
		cancel:       cancel,
		isActive:     true,
		lastActivity: time.Now(),
		maxErrors:    10,
	}

	// Store session
	h.mu.Lock()
	h.sessions[sessionID] = session
	h.mu.Unlock()

	log.Printf("✅ Enhanced WebSocket connected for session %s", sessionID)

	// Initialize CDP connection
	if err := h.initializeCDPConnection(session); err != nil {
		log.Printf("❌ Failed to initialize CDP connection: %v", err)
		h.cleanupSession(sessionID)
		return err
	}

	// Start session management goroutines
	go h.handleMessages(session)
	go h.streamScreenshots(session)
	go h.monitorSession(session)

	// Send initial connection confirmation
	h.sendMessage(session, map[string]interface{}{
		"type":      "connected",
		"message":   "Enhanced WebSocket streaming started",
		"sessionId": sessionID,
		"timestamp": time.Now().Unix(),
	})

	// Keep the connection alive
	select {
	case <-session.ctx.Done():
		log.Printf("🔌 Enhanced WebSocket session %s context cancelled", sessionID)
	}

	h.cleanupSession(sessionID)
	return nil
}

// initializeCDPConnection sets up the CDP connection for the session
func (h *EnhancedWebSocketHandler) initializeCDPConnection(session *EnhancedSessionManager) error {
	// Get CDP endpoint from global sessions (you'll need to implement this based on your session management)
	cdpURL := fmt.Sprintf("ws://127.0.0.1:9232/devtools/page/%s", "7642546E7E8803DCDE25E6A13F458809") // Placeholder - replace with actual page ID
	
	log.Printf("🔗 Initializing enhanced CDP connection to: %s", cdpURL)
	
	// Connect to CDP with enhanced service
	if err := session.cdpService.Connect(cdpURL); err != nil {
		return fmt.Errorf("failed to connect to CDP: %w", err)
	}

	log.Printf("✅ Enhanced CDP connection established for session %s", session.sessionID)
	return nil
}

// handleMessages processes incoming WebSocket messages
func (h *EnhancedWebSocketHandler) handleMessages(session *EnhancedSessionManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Message handler panic for session %s: %v", session.sessionID, r)
		}
		session.cancel()
	}()

	// Set initial read deadline
	session.wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	
	// Set pong handler
	session.wsConn.SetPongHandler(func(string) error {
		session.wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	log.Printf("🎧 Enhanced message handler started for session %s", session.sessionID)

	for {
		select {
		case <-session.ctx.Done():
			return
		default:
		}

		// Read message with timeout
		messageType, message, err := session.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ Enhanced WebSocket read error for session %s: %v", session.sessionID, err)
			}
			return
		}

		// Update activity time
		session.lastActivity = time.Now()

		// Process message based on type
		switch messageType {
		case websocket.TextMessage:
			log.Printf("🔍 Enhanced WebSocket received message type: %d, length: %d", messageType, len(message))
			if err := h.processTextMessage(session, message); err != nil {
				log.Printf("❌ Failed to process text message: %v", err)
				session.errorCount++
				if session.errorCount > session.maxErrors {
					log.Printf("⚠️ Too many errors for session %s, closing", session.sessionID)
					return
				}
			}
		case websocket.PingMessage:
			// Respond to ping
			session.wsConn.WriteMessage(websocket.PongMessage, nil)
		}
	}
}

// processTextMessage processes incoming text messages
func (h *EnhancedWebSocketHandler) processTextMessage(session *EnhancedSessionManager, message []byte) error {
	log.Printf("📨 Enhanced WebSocket text message: %s", string(message))

	// Parse message
	var msgData map[string]interface{}
	if err := json.Unmarshal(message, &msgData); err != nil {
		return fmt.Errorf("failed to parse message JSON: %w", err)
	}

	log.Printf("🔍 Parsed message data: %+v", msgData)

	msgType, ok := msgData["type"].(string)
	if !ok {
		return fmt.Errorf("message missing type field")
	}

	log.Printf("🔍 Message type: %s", msgType)

	switch msgType {
	case "handshake":
		return h.handleHandshake(session, msgData)
	case "action":
		return h.handleUserAction(session, msgData)
	default:
		log.Printf("⚠️ Unknown message type: %s", msgType)
		return nil
	}
}

// handleHandshake processes handshake messages
func (h *EnhancedWebSocketHandler) handleHandshake(session *EnhancedSessionManager, msgData map[string]interface{}) error {
	log.Printf("🤝 Enhanced WebSocket handshake received for session %s", session.sessionID)
	
	// Send handshake acknowledgment
	return h.sendMessage(session, map[string]interface{}{
		"type":      "handshake_ack",
		"sessionId": session.sessionID,
		"timestamp": time.Now().Unix(),
	})
}

// handleUserAction processes user action messages
func (h *EnhancedWebSocketHandler) handleUserAction(session *EnhancedSessionManager, msgData map[string]interface{}) error {
	action, ok := msgData["action"].(string)
	if !ok {
		return fmt.Errorf("action message missing action field")
	}

	data, ok := msgData["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("action message missing data field")
	}

	log.Printf("🎮 Enhanced action detected: %s", action)
	log.Printf("🎮 Processing enhanced user interaction: action=%s, session=%s, data=%+v", action, session.sessionID, data)

	// Process the action with enhanced error handling
	if err := h.processUserActionWithRecovery(session, action, data); err != nil {
		log.Printf("⚠️ Failed to handle enhanced user interaction: %v", err)
		
		// Send error response
		h.sendMessage(session, map[string]interface{}{
			"type":      "action_error",
			"message":   err.Error(),
			"timestamp": time.Now().Unix(),
		})
		return err
	}

	// Send success acknowledgment
	return h.sendMessage(session, map[string]interface{}{
		"type":      "action_ack",
		"action":    action,
		"timestamp": time.Now().Unix(),
	})
}

// processUserActionWithRecovery processes user actions with automatic recovery
func (h *EnhancedWebSocketHandler) processUserActionWithRecovery(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	maxRetries := 3
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("🔄 Retrying enhanced user action %s (attempt %d/%d)", action, attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second) // Progressive delay
		}
		
		err := h.processUserActionOnce(session, action, data)
		if err == nil {
			log.Printf("✅ Successfully processed enhanced user interaction: %s", action)
			return nil
		}
		
		log.Printf("❌ Enhanced user action attempt %d failed: %v", attempt+1, err)
		
		// Check if we should retry based on error type
		errorStr := err.Error()
		if attempt < maxRetries-1 && (
			fmt.Sprintf("%v", errorStr) == "timeout" ||
			fmt.Sprintf("%v", errorStr) == "connection reset" ||
			fmt.Sprintf("%v", errorStr) == "broken pipe") {
			
			log.Printf("🔄 Recoverable error detected, will retry")
			continue
		}
		
		if attempt == maxRetries-1 {
			return fmt.Errorf("enhanced user interaction failed after %d attempts: %w", maxRetries, err)
		}
	}
	
	return fmt.Errorf("enhanced user interaction failed after %d attempts", maxRetries)
}

// processUserActionOnce performs a single user action attempt
func (h *EnhancedWebSocketHandler) processUserActionOnce(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	switch action {
	case "mousedown", "mouseup", "click":
		return h.handleMouseAction(session, action, data)
	case "key", "keydown", "keyup":
		return h.handleKeyAction(session, action, data)
	case "scroll":
		return h.handleScrollAction(session, action, data)
	case "type":
		return h.handleTypeAction(session, action, data)
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

// handleMouseAction processes mouse actions
func (h *EnhancedWebSocketHandler) handleMouseAction(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	x, ok := data["x"].(float64)
	if !ok {
		return fmt.Errorf("mouse action missing x coordinate")
	}
	
	y, ok := data["y"].(float64)
	if !ok {
		return fmt.Errorf("mouse action missing y coordinate")
	}
	
	log.Printf("🖱️ Enhanced mouse action: %s at coordinates (%.0f, %.0f)", action, x, y)
	
	// Use the enhanced CDP service
	return session.cdpService.Click(int(x), int(y))
}

// handleKeyAction processes key actions
func (h *EnhancedWebSocketHandler) handleKeyAction(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	key, ok := data["key"].(string)
	if !ok {
		return fmt.Errorf("key action missing key field")
	}
	
	log.Printf("⌨️ Enhanced key action: %s, key: '%s'", action, key)
	
	// Use the enhanced CDP service
	return session.cdpService.PressKey(key)
}

// handleScrollAction processes scroll actions
func (h *EnhancedWebSocketHandler) handleScrollAction(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	// Implementation for scroll actions
	log.Printf("📜 Enhanced scroll action: %s, data: %+v", action, data)
	// Add scroll implementation here
	return nil
}

// handleTypeAction processes type actions
func (h *EnhancedWebSocketHandler) handleTypeAction(session *EnhancedSessionManager, action string, data map[string]interface{}) error {
	text, ok := data["text"].(string)
	if !ok {
		return fmt.Errorf("type action missing text field")
	}
	
	log.Printf("📝 Enhanced type action: '%s'", text)
	
	// Type each character
	for _, char := range text {
		if err := session.cdpService.PressKey(string(char)); err != nil {
			return fmt.Errorf("failed to type character '%c': %w", char, err)
		}
		time.Sleep(50 * time.Millisecond) // Delay between characters
	}
	
	return nil
}

// streamScreenshots continuously streams screenshots to the WebSocket
func (h *EnhancedWebSocketHandler) streamScreenshots(session *EnhancedSessionManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Screenshot streamer panic for session %s: %v", session.sessionID, r)
		}
	}()

	log.Printf("📹 Enhanced screenshot streaming started for session %s", session.sessionID)

	ticker := time.NewTicker(500 * time.Millisecond) // 2 FPS
	defer ticker.Stop()

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-ticker.C:
			if err := h.captureAndSendScreenshot(session); err != nil {
				log.Printf("⚠️ Enhanced screenshot capture failed: %v", err)
				// Don't break the loop, just log the error and continue
			}
		}
	}
}

// captureAndSendScreenshot captures and sends a screenshot
func (h *EnhancedWebSocketHandler) captureAndSendScreenshot(session *EnhancedSessionManager) error {
	// Capture screenshot using enhanced CDP service
	screenshotData, err := session.cdpService.TakeScreenshot()
	if err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Encode screenshot as base64
	base64Data := base64.StdEncoding.EncodeToString(screenshotData)

	// Send screenshot via WebSocket
	session.wsConnMutex.Lock()
	defer session.wsConnMutex.Unlock()

	if session.wsConn == nil {
		return fmt.Errorf("WebSocket connection not available")
	}

	session.wsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := session.wsConn.WriteMessage(websocket.BinaryMessage, []byte(base64Data)); err != nil {
		return fmt.Errorf("failed to send screenshot: %w", err)
	}

	return nil
}

// monitorSession monitors session health and performs cleanup
func (h *EnhancedWebSocketHandler) monitorSession(session *EnhancedSessionManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Session monitor panic for session %s: %v", session.sessionID, r)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-session.ctx.Done():
			return
		case <-ticker.C:
			// Check session health
			if time.Since(session.lastActivity) > 5*time.Minute {
				log.Printf("⚠️ Session %s inactive for too long, cleaning up", session.sessionID)
				session.cancel()
				return
			}
			
			// Check CDP connection status
			isConnected, _, failures := session.cdpService.GetConnectionStatus()
			if !isConnected || failures > 5 {
				log.Printf("⚠️ CDP connection unhealthy for session %s (connected: %v, failures: %d)", 
					session.sessionID, isConnected, failures)
			}
		}
	}
}

// sendMessage sends a message to the WebSocket client
func (h *EnhancedWebSocketHandler) sendMessage(session *EnhancedSessionManager, message interface{}) error {
	session.wsConnMutex.Lock()
	defer session.wsConnMutex.Unlock()

	if session.wsConn == nil {
		return fmt.Errorf("WebSocket connection not available")
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	session.wsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := session.wsConn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// cleanupSession cleans up session resources
func (h *EnhancedWebSocketHandler) cleanupSession(sessionID string) {
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	if exists {
		delete(h.sessions, sessionID)
	}
	h.mu.Unlock()

	if !exists {
		return
	}

	log.Printf("🧹 Cleaning up enhanced session %s", sessionID)

	// Cancel context
	session.cancel()

	// Close CDP service
	if session.cdpService != nil {
		session.cdpService.Close()
	}

	// Close WebSocket connection
	if session.wsConn != nil {
		session.wsConn.Close()
	}

	// Close message queue
	close(session.messageQueue)

	log.Printf("✅ Enhanced session %s cleaned up", sessionID)
}

// GetSessionCount returns the number of active sessions
func (h *EnhancedWebSocketHandler) GetSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}