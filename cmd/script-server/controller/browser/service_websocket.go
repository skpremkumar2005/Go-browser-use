package browser

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket Management

func (ws *WebSocketManager) run() {
	for {
		select {
		case client := <-ws.register:
			ws.mutex.Lock()
			ws.clients[client] = true
			ws.mutex.Unlock()
			log.Printf("📡 WebSocket client connected. Total clients: %d", len(ws.clients))

		case client := <-ws.unregister:
			ws.mutex.Lock()
			if _, ok := ws.clients[client]; ok {
				delete(ws.clients, client)
				close(client.Send)
			}
			ws.mutex.Unlock()
			log.Printf("📡 WebSocket client disconnected. Total clients: %d", len(ws.clients))

		case message := <-ws.broadcast:
			ws.mutex.RLock()
			for client := range ws.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(ws.clients, client)
				}
			}
			ws.mutex.RUnlock()
		}
	}
}

func (ws *WebSocketManager) broadcastTaskUpdate(taskId, status string) {
	message := map[string]interface{}{
		"type":      "task_update",
		"taskId":    taskId,
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Failed to marshal task update message: %v", err)
		return
	}

	ws.broadcast <- messageBytes
	log.Printf("📡 Broadcasted task update: %s -> %s", taskId, status)
}

func (c *WebSocketClient) ReadPump() {
	defer func() {
		wsManager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var wsMessage map[string]interface{}
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("⚠️ Failed to parse WebSocket message: %v", err)
			continue
		}

		if messageType, ok := wsMessage["type"].(string); ok && messageType == "action" {
			c.handleActionMessage(wsMessage)
		}
	}
}

func (c *WebSocketClient) handleActionMessage(message map[string]interface{}) {
	taskId, ok := message["taskId"].(string)
	if !ok {
		log.Printf("⚠️ Missing taskId in WebSocket action message")
		return
	}

	action, ok := message["action"].(string)
	if !ok {
		log.Printf("⚠️ Missing action in WebSocket action message")
		return
	}

	data, ok := message["data"].(map[string]interface{})
	if !ok {
		log.Printf("⚠️ Missing data in WebSocket action message")
		return
	}

	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[taskId]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		log.Printf("⚠️ Session not found for WebSocket action: %s", taskId)
		return
	}

	browserSession := browserManager.GetSession(session.BrowserID)
	if browserSession == nil {
		log.Printf("⚠️ Browser session not found for WebSocket action: %s", session.BrowserID)
		return
	}

	cdpClient := NewCDPClient(browserSession.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		log.Printf("❌ Failed to connect to CDP for WebSocket action: %v", err)
		return
	}
	defer cdpClient.Close()

	var err error
	
	browserSession.LastUserInteraction = time.Now()
	
	switch action {
	case "click":
		if x, xOk := data["x"].(float64); xOk {
			if y, yOk := data["y"].(float64); yOk {
				log.Printf("🖱️ WebSocket click: (%.0f, %.0f)", x, y)
				err = cdpClient.Click(x, y)
			}
		}
	case "type":
		if text, ok := data["text"].(string); ok {
			log.Printf("⌨️ WebSocket type: '%s'", text)
			err = cdpClient.TypeText(text)
		}
	case "key":
		if key, ok := data["key"].(string); ok {
			log.Printf("⌨️ WebSocket key: '%s'", key)
			err = cdpClient.PressKey(key)
		}
	case "scroll":
		if deltaX, xOk := data["deltaX"].(float64); xOk {
			if deltaY, yOk := data["deltaY"].(float64); yOk {
				log.Printf("📜 WebSocket scroll: deltaX=%.0f deltaY=%.0f", deltaX, deltaY)
				err = cdpClient.Scroll(deltaX, deltaY)
			}
		}
	}

	if err != nil {
		log.Printf("❌ WebSocket action failed: %v", err)
	} else {
		log.Printf("✅ WebSocket action executed successfully: %s", action)
		browserManager.updateUserInteraction(session.BrowserID)
	}
}

func (c *WebSocketClient) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
