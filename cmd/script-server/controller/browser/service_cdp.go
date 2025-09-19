package browser

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go-webrtc/cmd/internal/stealth"
)

// CDPClient functions

func NewCDPClient(wsURL string) *CDPClient {
	ctx, cancel := context.WithCancel(context.Background())
	client := &CDPClient{
		wsURL:         wsURL,
		ctx:           ctx,
		cancel:        cancel,
		requestID:     1,
		commandQueue:  make(chan CDPCommand, 100), // Buffer for 100 commands
		responses:     make(map[int]chan CDPResponse),
		queueWorkers:  2,
		maxQueueSize:  100,
		isProcessing:  false,
	}
	
	// Start command queue workers
	for i := 0; i < client.queueWorkers; i++ {
		go client.processCommandQueue()
	}
	
	return client
}

func (c *CDPClient) Connect() error {
    // Fast path: if already connected, do nothing
    c.mutex.RLock()
    if c.conn != nil {
        c.mutex.RUnlock()
        return nil
    }
    c.mutex.RUnlock()

	u, err := url.Parse(c.wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second // Increased timeout

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to CDP: %v", err)
	}

	// More generous timeouts for concurrent operations
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))  // Increased from 30s
	conn.SetWriteDeadline(time.Now().Add(20 * time.Second)) // Increased from 10s

	c.mutex.Lock()
	c.conn = conn
	c.mutex.Unlock()

	if !c.initialized {
		_, err = c.SendCommand("Page.enable", map[string]interface{}{})
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to enable Page domain: %v", err)
		}
		c.pageEnabled = true
		c.initialized = true
		log.Printf("🔗 Connected to CDP at %s (Page domain enabled)", c.wsURL)
	} else {
		log.Printf("🔗 Reconnected to CDP at %s", c.wsURL)
	}

	return nil
}

func (c *CDPClient) Close() error {
	c.cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// processCommandQueue handles queued CDP commands to prevent overload
func (c *CDPClient) processCommandQueue() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case cmd := <-c.commandQueue:
			c.executeCommand(cmd)
		}
	}
}

// executeCommand executes a single CDP command
func (c *CDPClient) executeCommand(cmd CDPCommand) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil {
		cmd.Response <- CDPResponse{Error: fmt.Errorf("CDP connection not established")}
		return
	}

	// Add delay based on priority (high priority = less delay)
	switch cmd.Priority {
	case 2: // High priority (screenshots, interactions)
		time.Sleep(25 * time.Millisecond)
	case 1: // Normal priority
		time.Sleep(50 * time.Millisecond)
	default: // Low priority
		time.Sleep(100 * time.Millisecond)
	}

	command := map[string]interface{}{
		"id":     cmd.ID,
		"method": cmd.Method,
		"params": cmd.Params,
	}

	// Set timeouts based on command type
	writeTimeout := 20 * time.Second
	readTimeout := 60 * time.Second
	
	if cmd.Method == "Page.captureScreenshot" {
		writeTimeout = 10 * time.Second
		readTimeout = 30 * time.Second
	}

	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.conn.WriteJSON(command); err != nil {
		cmd.Response <- CDPResponse{Error: fmt.Errorf("failed to send command: %v", err)}
		return
	}

	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	for {
		var response map[string]interface{}
		if err := c.conn.ReadJSON(&response); err != nil {
			cmd.Response <- CDPResponse{Error: fmt.Errorf("failed to read response: %v", err)}
			return
		}

		// CDP events won't have an 'id'; responses will.
		if idVal, ok := response["id"]; ok {
			var id int
			switch v := idVal.(type) {
			case float64:
				id = int(v)
			case int:
				id = v
			default:
				id = 0
			}
			if id != cmd.ID {
				// Unexpected response ID (shouldn't happen with serialized access); continue reading
				continue
			}

			if errorData, exists := response["error"]; exists {
				cmd.Response <- CDPResponse{Error: fmt.Errorf("CDP error: %v", errorData)}
				return
			}

			if result, exists := response["result"]; exists {
				if resultMap, ok := result.(map[string]interface{}); ok {
					cmd.Response <- CDPResponse{Result: resultMap}
					return
				}
			}

			// Some commands may return without nested result, pass through
			cmd.Response <- CDPResponse{Result: response}
			return
		}

		// No id -> it's an async event; ignore and keep reading
		continue
	}
}

func (c *CDPClient) SendCommand(method string, params map[string]interface{}) (map[string]interface{}, error) {
	return c.SendCommandWithPriority(method, params, 1) // Default normal priority
}

func (c *CDPClient) SendCommandWithPriority(method string, params map[string]interface{}, priority int) (map[string]interface{}, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("CDP connection not established")
	}

	// Generate unique request ID
	c.mutex.Lock()
	c.requestID++
	requestID := c.requestID
	c.mutex.Unlock()

	// Create response channel
	responseChan := make(chan CDPResponse, 1)

	// Create command
	cmd := CDPCommand{
		ID:       requestID,
		Method:   method,
		Params:   params,
		Response: responseChan,
		Priority: priority,
	}

	// Queue the command (with timeout to prevent blocking)
	select {
	case c.commandQueue <- cmd:
		// Command queued successfully
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("command queue full, timeout after 5 seconds")
	}

	// Wait for response (with timeout)
	select {
	case response := <-responseChan:
		if response.Error != nil {
			return nil, response.Error
		}
		return response.Result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("command timeout after 30 seconds")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("CDP client context cancelled")
	}
}

func (c *CDPClient) CaptureScreenshot() ([]byte, error) {
	if !c.initialized {
		return nil, fmt.Errorf("CDP client not initialized")
	}

	params := map[string]interface{}{
		"format":      "jpeg",
		"quality":     95,
		"fromSurface": true,
	}

	response, err := c.SendCommandWithPriority("Page.captureScreenshot", params, 2) // High priority
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %v", err)
	}

	var dataStr string
	var found bool

	if result, exists := response["result"]; exists {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if data, exists := resultMap["data"]; exists {
				if dataStr, ok = data.(string); ok {
					found = true
				}
			}
		}
	}

	if !found {
		if data, exists := response["data"]; exists {
			if str, ok := data.(string); ok {
				dataStr = str
				found = true
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no valid data found in screenshot response")
	}

	imageData, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot: %v", err)
	}

	if len(imageData) < 1000 {
		return nil, fmt.Errorf("screenshot too small (%d bytes), likely blank", len(imageData))
	}

	return imageData, nil
}

func (c *CDPClient) CaptureScreenshotWithRetry(maxRetries int) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		imageData, err := c.CaptureScreenshot()
		if err == nil {
			return imageData, nil
		}

		lastErr = err
		log.Printf("⚠️ Screenshot attempt %d/%d failed: %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * 100 * time.Millisecond
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed to capture screenshot after %d attempts: %v", maxRetries, lastErr)
}

func (c *CDPClient) InjectStealthScript() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	stealthScript := getStealthJavaScript()

	params := map[string]interface{}{
		"source": stealthScript,
	}

	_, err := c.SendCommand("Runtime.evaluate", params)
	if err != nil {
		return fmt.Errorf("failed to inject stealth script: %v", err)
	}

	log.Printf("🥷 Stealth JavaScript injected successfully")
	return nil
}

func (c *CDPClient) SetupStealthEnvironment() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	domains := []string{"Runtime", "Network", "Page", "Fetch"}
	for _, domain := range domains {
		_, err := c.SendCommand(domain+".enable", map[string]interface{}{})
		if err != nil {
			log.Printf("⚠️ Failed to enable %s domain: %v", domain, err)
		}
	}

	_, err := c.SendCommand("Fetch.enable", map[string]interface{}{
		"patterns": []map[string]interface{}{
			{
				"urlPattern":   "*",
				"requestStage": "Request",
			},
		},
	})
	if err != nil {
		log.Printf("⚠️ Failed to enable request interception: %v", err)
	}

	networkConditions := stealth.GetStealthNetworkEmulation()
	_, err = c.SendCommand("Network.emulateNetworkConditions", networkConditions)
	if err != nil {
		log.Printf("⚠️ Failed to set network conditions: %v", err)
	}

	headers := stealth.GetUltraStealthHeaders()
	_, err = c.SendCommand("Network.setUserAgentOverride", map[string]interface{}{
		"userAgent":      headers["User-Agent"],
		"acceptLanguage": headers["Accept-Language"],
		"platform":       "Win32",
		"userAgentMetadata": map[string]interface{}{
			"brands": []map[string]interface{}{
				{"brand": "Not_A Brand", "version": "8"},
				{"brand": "Chromium", "version": "120"},
				{"brand": "Google Chrome", "version": "120"},
			},
			"fullVersion":     "120.0.0.0",
			"platform":        "Windows",
			"platformVersion": "10.0.0",
			"architecture":    "x86",
			"model":           "",
			"mobile":          false,
			"bitness":         "64",
		},
	})
	if err != nil {
		log.Printf("⚠️ Failed to set user agent override: %v", err)
	}

	if err := c.InjectStealthScript(); err != nil {
		return err
	}

	viewportParams := map[string]interface{}{
		"width":             1920,
		"height":            1080,
		"deviceScaleFactor": 1,
		"mobile":            false,
	}

	_, err = c.SendCommand("Emulation.setDeviceMetricsOverride", viewportParams)
	if err != nil {
		log.Printf("⚠️ Failed to set viewport: %v", err)
	}

	timezoneParams := map[string]interface{}{
		"timezoneId": "America/New_York",
	}

	_, err = c.SendCommand("Emulation.setTimezoneOverride", timezoneParams)
	if err != nil {
		log.Printf("⚠️ Failed to set timezone: %v", err)
	}

	localeParams := map[string]interface{}{
		"locale": "en-US",
	}

	_, err = c.SendCommand("Emulation.setLocaleOverride", localeParams)
	if err != nil {
		log.Printf("⚠️ Failed to set locale: %v", err)
	}

	log.Printf("🛡️ Stealth environment configured successfully")
	return nil
}

func (c *CDPClient) SetupPageNavigationStealth() error {
	if !c.initialized {
		return fmt.Errorf("CDP client not initialized")
	}

	stealthScript := getStealthJavaScript()
	params := map[string]interface{}{
		"source": stealthScript,
	}

	_, err := c.SendCommand("Page.addScriptToEvaluateOnNewDocument", params)
	if err != nil {
		return fmt.Errorf("failed to add stealth script to new documents: %v", err)
	}

	_, err = c.SendCommand("Runtime.evaluate", params)
	if err != nil {
		log.Printf("⚠️ Failed to inject stealth script on current page: %v", err)
	}

	log.Printf("🔄 Page navigation stealth listener activated")
	return nil
}

func (c *CDPClient) Reconnect() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.initialized = false
	c.pageEnabled = false

	return c.Connect()
}

func (c *CDPClient) Click(x, y float64) error {
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	jsCode := fmt.Sprintf(`
		var event = new MouseEvent('click', {
			view: window,
			bubbles: true,
			cancelable: true,
			clientX: %f,
			clientY: %f
		});
		var element = document.elementFromPoint(%f, %f);
		if (element) {
			element.dispatchEvent(event);
		}
	`, x, y, x, y)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute click: %v", err)
	}

	log.Printf("🖱️ Clicked at coordinates (%.0f, %.0f) via JavaScript", x, y)
	return nil
}

func (c *CDPClient) TypeText(text string) error {
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	escapedText := strings.ReplaceAll(text, `\`, `\\`)
	escapedText = strings.ReplaceAll(escapedText, `"`, `\"`)
	escapedText = strings.ReplaceAll(escapedText, "\n", `\n`)
	escapedText = strings.ReplaceAll(escapedText, "\r", `\r`)

	jsCode := fmt.Sprintf(`
		var activeElement = document.activeElement;
		if (activeElement && (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA' || activeElement.contentEditable === 'true')) {
			var start = activeElement.selectionStart || 0;
			var end = activeElement.selectionEnd || 0;
			var value = activeElement.value || activeElement.textContent || '';
			var newValue = value.substring(0, start) + "%s" + value.substring(end);
			if (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA') {
				activeElement.value = newValue;
			} else {
				activeElement.textContent = newValue;
			}
			activeElement.selectionStart = activeElement.selectionEnd = start + %d;
			activeElement.dispatchEvent(new Event('input', { bubbles: true }));
			activeElement.dispatchEvent(new Event('change', { bubbles: true }));
		} else {
			var input = document.querySelector('input, textarea, [contenteditable="true"]');
			if (input) {
				input.focus();
				input.value = input.value + "%s";
				input.dispatchEvent(new Event('input', { bubbles: true }));
			}
		}
	`, escapedText, len(text), escapedText)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute type: %v", err)
	}

	log.Printf("⌨️ Typed text: %s via JavaScript", text)
	return nil
}

func (c *CDPClient) PressKey(key string) error {
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	keyCodeMap := map[string]int{
		"Enter":      13,
		"Tab":        9,
		"Escape":     27,
		"Backspace":  8,
		"Delete":     46,
		"ArrowUp":    38,
		"ArrowDown":  40,
		"ArrowLeft":  37,
		"ArrowRight": 39,
		"Space":      32,
	}

	keyCode, exists := keyCodeMap[key]
	if !exists {
		keyCode = int(key[0])
	}

	jsCode := fmt.Sprintf(`
		var activeElement = document.activeElement;
		var event = new KeyboardEvent('keydown', {
			key: '%s',
			code: '%s',
			keyCode: %d,
			which: %d,
			bubbles: true,
			cancelable: true
		});
		if (activeElement) {
			activeElement.dispatchEvent(event);
		} else {
			document.dispatchEvent(event);
		}
		
		var eventUp = new KeyboardEvent('keyup', {
			key: '%s',
			code: '%s',
			keyCode: %d,
			which: %d,
			bubbles: true,
			cancelable: true
		});
		if (activeElement) {
			activeElement.dispatchEvent(eventUp);
		} else {
			document.dispatchEvent(eventUp);
		}
	`, key, key, keyCode, keyCode, key, key, keyCode, keyCode)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute key press: %v", err)
	}

	log.Printf("⌨️ Pressed key: %s via JavaScript", key)
	return nil
}

func (c *CDPClient) Scroll(deltaX, deltaY float64) error {
	_, err := c.SendCommand("Runtime.enable", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to enable Runtime domain: %v", err)
	}

	jsCode := fmt.Sprintf(`
		window.scrollBy(%f, %f);
	`, deltaX, deltaY)

	_, err = c.SendCommand("Runtime.evaluate", map[string]interface{}{
		"expression": jsCode,
	})
	if err != nil {
		return fmt.Errorf("failed to execute scroll: %v", err)
	}

	log.Printf("📜 Scrolled by (%.0f, %.0f) via JavaScript", deltaX, deltaY)
	return nil
}
