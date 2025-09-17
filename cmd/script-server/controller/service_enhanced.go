package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EnhancedService provides robust CDP client with advanced timeout handling and connection management
type EnhancedService struct {
	cdpURL       string
	conn         *websocket.Conn
	connMutex    sync.RWMutex
	requestID    int64
	requestMutex sync.Mutex
	sessionID    string
	
	// Enhanced configuration
	dialTimeout      time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	pingInterval     time.Duration
	pongTimeout      time.Duration
	reconnectAttempts int
	reconnectDelay   time.Duration
	
	// Connection state management
	lastConnectTime  time.Time
	consecutiveFailures int
	isConnected      bool
	
	// Buffer management
	readBufferSize   int
	writeBufferSize  int
}

// NewEnhancedService creates a new enhanced CDP service with robust timeout handling
func NewEnhancedService() *EnhancedService {
	return &EnhancedService{
		dialTimeout:       30 * time.Second,  // Increased from default
		readTimeout:       60 * time.Second,  // Much longer read timeout
		writeTimeout:      30 * time.Second,  // Longer write timeout
		pingInterval:      30 * time.Second,  // Keep connection alive
		pongTimeout:       10 * time.Second,  // Pong response timeout
		reconnectAttempts: 5,                 // Maximum reconnection attempts
		reconnectDelay:    2 * time.Second,   // Base reconnection delay
		readBufferSize:    65536,             // 64KB read buffer
		writeBufferSize:   65536,             // 64KB write buffer
	}
}

// Connect establishes a CDP connection with enhanced error handling and timeouts
func (s *EnhancedService) Connect(cdpURL string) error {
	s.cdpURL = cdpURL
	
	for attempt := 0; attempt <= s.reconnectAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(float64(s.reconnectDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			log.Printf("🔄 CDP connection attempt %d/%d, waiting %v", attempt+1, s.reconnectAttempts+1, delay)
			time.Sleep(delay)
		}
		
		if err := s.connectOnce(); err != nil {
			log.Printf("❌ CDP connection attempt %d failed: %v", attempt+1, err)
			s.consecutiveFailures++
			continue
		}
		
		log.Printf("✅ CDP connection established successfully after %d attempts", attempt+1)
		s.consecutiveFailures = 0
		s.isConnected = true
		s.lastConnectTime = time.Now()
		
		// Start keep-alive pinger
		go s.keepAlive()
		
		return nil
	}
	
	return fmt.Errorf("failed to establish CDP connection after %d attempts", s.reconnectAttempts+1)
}

// connectOnce performs a single connection attempt with proper timeout handling
func (s *EnhancedService) connectOnce() error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	
	// Close existing connection if any
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	
	// Create custom dialer with enhanced settings
	dialer := &websocket.Dialer{
		HandshakeTimeout: s.dialTimeout,
		ReadBufferSize:   s.readBufferSize,
		WriteBufferSize:  s.writeBufferSize,
		EnableCompression: false, // Disable compression to avoid buffer issues
	}
	
	// Create connection context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.dialTimeout)
	defer cancel()
	
	// Connect to CDP WebSocket
	conn, _, err := dialer.DialContext(ctx, s.cdpURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial CDP WebSocket: %w", err)
	}
	
	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(s.readTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	
	// Set pong handler for keep-alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		return nil
	})
	
	s.conn = conn
	
	// Enable Page domain for CDP operations
	if err := s.enablePageDomain(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	
	log.Printf("✅ CDP connection established and Page domain enabled")
	return nil
}

// keepAlive maintains the connection with periodic pings
func (s *EnhancedService) keepAlive() {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		s.connMutex.RLock()
		if s.conn == nil || !s.isConnected {
			s.connMutex.RUnlock()
			return
		}
		conn := s.conn
		s.connMutex.RUnlock()
		
		// Send ping
		conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("⚠️ Keep-alive ping failed: %v", err)
			s.handleConnectionFailure()
			return
		}
	}
}

// handleConnectionFailure manages connection failures and triggers reconnection
func (s *EnhancedService) handleConnectionFailure() {
	s.connMutex.Lock()
	s.isConnected = false
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.connMutex.Unlock()
	
	log.Printf("🔄 Connection failure detected, attempting reconnection...")
	go func() {
		if err := s.Connect(s.cdpURL); err != nil {
			log.Printf("❌ Failed to reconnect CDP: %v", err)
		}
	}()
}

// enablePageDomain enables the Page domain for CDP operations
func (s *EnhancedService) enablePageDomain() error {
	enablePageReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Page.enable",
	}
	
	if err := s.sendRequestWithRetry(enablePageReq); err != nil {
		return fmt.Errorf("failed to enable Page domain: %w", err)
	}
	
	return nil
}

// sendRequestWithRetry sends a CDP request with automatic retry on failure
func (s *EnhancedService) sendRequestWithRetry(request map[string]interface{}) error {
	maxRetries := 3
	baseDelay := 1 * time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
			log.Printf("🔄 Retrying CDP request in %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			time.Sleep(delay)
		}
		
		err := s.sendRequest(request)
		if err == nil {
			return nil
		}
		
		// Check if it's a connection error
		if strings.Contains(err.Error(), "timeout") || 
		   strings.Contains(err.Error(), "connection reset") ||
		   strings.Contains(err.Error(), "broken pipe") {
			log.Printf("⚠️ Connection error detected, attempting reconnection: %v", err)
			if reconErr := s.Connect(s.cdpURL); reconErr != nil {
				log.Printf("❌ Reconnection failed: %v", reconErr)
				continue
			}
			continue
		}
		
		if attempt == maxRetries-1 {
			return fmt.Errorf("request failed after %d attempts: %w", maxRetries, err)
		}
	}
	
	return fmt.Errorf("request failed after %d attempts", maxRetries)
}

// sendRequest sends a CDP request with proper timeout handling
func (s *EnhancedService) sendRequest(request map[string]interface{}) error {
	s.connMutex.RLock()
	conn := s.conn
	if conn == nil || !s.isConnected {
		s.connMutex.RUnlock()
		return fmt.Errorf("CDP connection not available")
	}
	s.connMutex.RUnlock()
	
	// Serialize request
	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Set write deadline
	conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	
	// Send request
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		s.handleConnectionFailure()
		return fmt.Errorf("failed to send CDP request: %w", err)
	}
	
	// Read response with timeout
	conn.SetReadDeadline(time.Now().Add(s.readTimeout))
	
	_, responseData, err := conn.ReadMessage()
	if err != nil {
		s.handleConnectionFailure()
		return fmt.Errorf("failed to read CDP response: %w", err)
	}
	
	// Parse response to check for errors
	var response struct {
		ID    int64                  `json:"id"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	
	if err := json.Unmarshal(responseData, &response); err != nil {
		return fmt.Errorf("failed to parse CDP response: %w", err)
	}
	
	if response.Error != nil {
		return fmt.Errorf("CDP error %d: %s", response.Error.Code, response.Error.Message)
	}
	
	return nil
}

// getNextRequestID generates a unique request ID
func (s *EnhancedService) getNextRequestID() int64 {
	s.requestMutex.Lock()
	defer s.requestMutex.Unlock()
	s.requestID++
	return s.requestID
}

// Click performs a mouse click with enhanced error handling
func (s *EnhancedService) Click(x, y int) error {
	// First try to ensure page is active
	if err := s.ensurePageActiveEnhanced(); err != nil {
		log.Printf("⚠️ Page activation failed, continuing with click: %v", err)
	}
	
	// Perform mouse press
	mouseDownReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Input.dispatchMouseEvent",
		"params": map[string]interface{}{
			"type":   "mousePressed",
			"x":      x,
			"y":      y,
			"button": "left",
			"clickCount": 1,
		},
	}
	
	if err := s.sendRequestWithRetry(mouseDownReq); err != nil {
		return fmt.Errorf("failed to send mouse press: %w", err)
	}
	
	// Small delay between press and release
	time.Sleep(50 * time.Millisecond)
	
	// Perform mouse release
	mouseUpReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Input.dispatchMouseEvent",
		"params": map[string]interface{}{
			"type":   "mouseReleased",
			"x":      x,
			"y":      y,
			"button": "left",
			"clickCount": 1,
		},
	}
	
	if err := s.sendRequestWithRetry(mouseUpReq); err != nil {
		return fmt.Errorf("failed to send mouse release: %w", err)
	}
	
	log.Printf("✅ Successfully clicked at coordinates (%d, %d)", x, y)
	return nil
}

// PressKey performs a key press with enhanced error handling
func (s *EnhancedService) PressKey(key string) error {
	// Ensure page is active
	if err := s.ensurePageActiveEnhanced(); err != nil {
		log.Printf("⚠️ Page activation failed, continuing with key press: %v", err)
	}
	
	// Send key down event
	keyDownReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Input.dispatchKeyEvent",
		"params": map[string]interface{}{
			"type": "keyDown",
			"key":  key,
		},
	}
	
	if err := s.sendRequestWithRetry(keyDownReq); err != nil {
		return fmt.Errorf("failed to send keyDown for '%s': %w", key, err)
	}
	
	// Small delay between key down and up
	time.Sleep(50 * time.Millisecond)
	
	// Send key up event
	keyUpReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Input.dispatchKeyEvent",
		"params": map[string]interface{}{
			"type": "keyUp",
			"key":  key,
		},
	}
	
	if err := s.sendRequestWithRetry(keyUpReq); err != nil {
		return fmt.Errorf("failed to send keyUp for '%s': %w", key, err)
	}
	
	log.Printf("✅ Successfully pressed key '%s'", key)
	return nil
}

// ensurePageActiveEnhanced ensures the page is active with enhanced error handling
func (s *EnhancedService) ensurePageActiveEnhanced() error {
	bringToFrontReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Page.bringToFront",
	}
	
	if err := s.sendRequestWithRetry(bringToFrontReq); err != nil {
		return fmt.Errorf("failed to bring page to front: %w", err)
	}
	
	return nil
}

// TakeScreenshot captures a screenshot with enhanced error handling
func (s *EnhancedService) TakeScreenshot() ([]byte, error) {
	screenshotReq := map[string]interface{}{
		"id":     s.getNextRequestID(),
		"method": "Page.captureScreenshot",
		"params": map[string]interface{}{
			"format":  "png",
			"quality": 80,
			"clip": map[string]interface{}{
				"x":      0,
				"y":      0,
				"width":  1920,
				"height": 1080,
				"scale":  1,
			},
		},
	}
	
	s.connMutex.RLock()
	conn := s.conn
	if conn == nil || !s.isConnected {
		s.connMutex.RUnlock()
		return nil, fmt.Errorf("CDP connection not available")
	}
	s.connMutex.RUnlock()
	
	// Send request
	requestData, err := json.Marshal(screenshotReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal screenshot request: %w", err)
	}
	
	conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		s.handleConnectionFailure()
		return nil, fmt.Errorf("failed to send screenshot request: %w", err)
	}
	
	// Read response with extended timeout for screenshot
	conn.SetReadDeadline(time.Now().Add(120 * time.Second)) // 2 minutes for screenshot
	_, responseData, err := conn.ReadMessage()
	if err != nil {
		s.handleConnectionFailure()
		return nil, fmt.Errorf("failed to read screenshot response: %w", err)
	}
	
	// Parse screenshot response
	var response struct {
		ID     int64 `json:"id"`
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, fmt.Errorf("failed to parse screenshot response: %w", err)
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("screenshot error %d: %s", response.Error.Code, response.Error.Message)
	}
	
	// Decode base64 screenshot data
	screenshotData := []byte(response.Result.Data)
	log.Printf("✅ Screenshot captured successfully (%d bytes)", len(screenshotData))
	
	return screenshotData, nil
}

// Close closes the CDP connection
func (s *EnhancedService) Close() error {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	
	s.isConnected = false
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// GetConnectionStatus returns the current connection status
func (s *EnhancedService) GetConnectionStatus() (bool, time.Time, int) {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()
	return s.isConnected, s.lastConnectTime, s.consecutiveFailures
}