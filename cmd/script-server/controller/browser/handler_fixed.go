package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// HandleWebSocketFixed - Corrected WebSocket handler with proper concurrent message handling
func (h *handler) HandleWebSocketFixed(c echo.Context) error {
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

	// Upgrade to WebSocket connection with error handling and proper buffer sizes
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
		ReadBufferSize:  16384, // Increase to 16KB to handle large frames
		WriteBufferSize: 16384, // Increase to 16KB to handle large frames
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

	// Create CDP client for screenshot capture and user interactions
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

	log.Printf("✅ CDP connection established for user interactions")

	// Send connection success message
	ws.WriteJSON(map[string]interface{}{
		"type": "connected",
		"sessionId": sessionID,
		"message": "WebSocket streaming started",
		"timestamp": time.Now().Unix(),
	})

	// Create channels for coordination
	done := make(chan bool)
	messageChan := make(chan []byte, 100) // Buffer for incoming messages

	// Start message handling in separate goroutine
	go h.handleWebSocketMessages(ws, cdpClient, sessionID, done, messageChan)

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
			done <- true
			return nil

		case <-done:
			log.Printf("📸 Message handler ended for WebSocket stream %s after %d frames", sessionID, frameCount)
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
				done <- true
				return nil
			}

			if connectionErrors >= maxErrors {
				log.Printf("❌ Too many connection errors (%d), stopping stream", connectionErrors)
				ws.WriteJSON(map[string]interface{}{
					"type": "stream_ended",
					"reason": "too_many_errors",
					"frames_sent": frameCount,
				})
				done <- true
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
				done <- true
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
						done <- true
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
				done <- true
				return nil
			}

			// Log progress
			if frameCount%400 == 0 {
				log.Printf("📹 Streamed %d frames via WebSocket for session %s", frameCount, sessionID)
			}
		}
	}
}

// handleWebSocketMessages - Separate goroutine for handling incoming WebSocket messages
func (h *handler) handleWebSocketMessages(ws *websocket.Conn, cdpClient *CDPClient, sessionID string, done chan bool, messageChan chan []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Message handler panic recovered: %v", r)
		}
		done <- true
	}()
	
	log.Printf("🎧 Message handler started for session %s", sessionID)
	
	for {
		// Handle incoming WebSocket messages (blocking read)
		messageType, message, err := ws.ReadMessage()
		
		if err != nil {
			// Check if connection is closing
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("📸 WebSocket connection closed normally")
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Continue on timeout
			}
			log.Printf("❌ WebSocket read error in message handler: %v", err)
			return
		}

		// Debug: Log ALL incoming messages
		log.Printf("🔍 WebSocket received message type: %d, length: %d", messageType, len(message))

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			// Handle text messages (commands, etc.)
			log.Printf("📨 WebSocket text message: %s", string(message))
			
			// Try to parse as JSON to see if it's an action message
			var msgData map[string]interface{}
			if err := json.Unmarshal(message, &msgData); err == nil {
				log.Printf("🔍 Parsed message data: %+v", msgData)
				if msgType, ok := msgData["type"].(string); ok {
					log.Printf("🔍 Message type: %s", msgType)
					if msgType == "action" {
						if action, actionOk := msgData["action"].(string); actionOk {
							// Handle user interaction with error recovery
							log.Printf("🎮 Action detected: %s", action)
							log.Printf("🎮 Received user interaction from frontend: action=%s, session=%s", msgData["action"], sessionID)
							
							// Attempt user interaction with recovery mechanism
							interactionErr := h.handleUserInteractionWithRecovery(cdpClient, message, sessionID)
							if interactionErr != nil {
								log.Printf("⚠️ Failed to handle user interaction: %v", interactionErr)
								// Send error response back to frontend
								errorResponse := map[string]interface{}{
									"type":    "action_error",
									"message": interactionErr.Error(),
									"timestamp": time.Now().Unix(),
								}
								ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
								if writeErr := ws.WriteJSON(errorResponse); writeErr != nil {
									log.Printf("❌ WebSocket write error: %v", writeErr)
									return
								}
							} else {
								log.Printf("✅ Successfully processed user interaction: %s", msgData["action"])
								// Send success acknowledgment
								ackResponse := map[string]interface{}{
									"type":      "action_ack",
									"action":    msgData["action"],
									"timestamp": time.Now().Unix(),
								}
								ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
								if writeErr := ws.WriteJSON(ackResponse); writeErr != nil {
									log.Printf("❌ WebSocket write error: %v", writeErr)
									return
								}
							}
							continue // Don't send the generic echo response
						}
					} else if msgType == "handshake" {
						log.Printf("🤝 WebSocket handshake received for session %s", sessionID)
						// Send handshake acknowledgment
						handshakeResponse := map[string]interface{}{
							"type": "handshake_ack",
							"sessionId": sessionID,
							"timestamp": time.Now().Unix(),
						}
						ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
						if err := ws.WriteJSON(handshakeResponse); err != nil {
							log.Printf("❌ WebSocket write error: %v", err)
							return
						}
						continue
					}
				}
			} else {
				log.Printf("❌ Failed to parse message as JSON: %v", err)
			}
			
			// Echo text messages back to client for debugging
			echoResponse := map[string]interface{}{
				"type":      "echo",
				"message":   string(message),
				"timestamp": time.Now().Unix(),
			}
			ws.SetWriteDeadline(time.Now().Add(1 * time.Second))
			if err := ws.WriteJSON(echoResponse); err != nil {
				log.Printf("❌ WebSocket write error: %v", err)
				return
			}

		case websocket.BinaryMessage:
			log.Printf("📦 Binary message received (length: %d bytes)", len(message))
			// Optionally handle binary messages here if needed

		case websocket.CloseMessage:
			log.Printf("🔌 WebSocket close message received")
			return

		default:
			log.Printf("❓ Unknown message type received: %d", messageType)
		}
	}
}

// handleUserInteractionWithRecovery - Handles user interactions with CDP client recovery
func (h *handler) handleUserInteractionWithRecovery(cdpClient *CDPClient, message []byte, sessionID string) error {
	// First attempt
	err := h.handleUserInteraction(cdpClient, message, sessionID)
	
	if err != nil {
		// Check if it's a CDP connection issue that can be recovered
		errStr := err.Error()
		if strings.Contains(errStr, "slice bounds out of range") ||
		   strings.Contains(errStr, "capacity") ||
		   strings.Contains(errStr, "websocket: close sent") ||
		   strings.Contains(errStr, "use of closed network connection") ||
		   strings.Contains(errStr, "RSV") {
			
			log.Printf("🔄 Attempting CDP connection recovery for user interaction")
			
			// Get fresh browser session to reconnect CDP
			browserSession, getErr := h.service.GetBrowserSession(sessionID)
			if getErr != nil {
				return fmt.Errorf("failed to get session for recovery: %w", getErr)
			}
			
			// Create new CDP client with proper buffers
			newCDPClient := NewCDPClient(browserSession.CDPEndpoint)
			if connErr := newCDPClient.Connect(); connErr != nil {
				return fmt.Errorf("failed to reconnect CDP for recovery: %w", connErr)
			}
			
			// Close old client and update reference
			cdpClient.Close()
			*cdpClient = *newCDPClient // Update the client in place
			
			// Retry the interaction with new connection
			retryErr := h.handleUserInteraction(cdpClient, message, sessionID)
			if retryErr != nil {
				return fmt.Errorf("user interaction failed after recovery: %w", retryErr)
			}
			
			log.Printf("✅ Successfully recovered CDP connection and processed user interaction")
			return nil
		}
		
		// If not a recoverable error, return the original error
		return err
	}
	
	return nil
}