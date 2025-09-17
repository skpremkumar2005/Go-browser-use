package browser

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go-webrtc/cmd/script-server/config"
)

// Helper functions for response creation

func CreateEnhancedTaskResponse(session *ScriptSession, browserSession *BrowserSession, r *http.Request) *EnhancedTaskResponse {
	baseURL := generateBaseURL(r)

	response := &EnhancedTaskResponse{
		Success:   true,
		TaskId:    session.ID,
		SessionId: session.ID,
		Status:    session.Status,
		Task:      session.Task,
		CreatedAt: session.CreatedAt,
		StreamingUrl: generateStreamingURL(baseURL, session.ID),
		WebSocketUrl: generateWebSocketURL(baseURL),
		Message:      "Task started successfully! Browser-use automation available at automation_url",
	}

	response.AutomationUrl = generateAutomationURL(baseURL, session.ID)

	if browserSession != nil {
		response.BrowserInfo = &BrowserInfo{
			BrowserId:   browserSession.ID,
			CDPEndpoint: browserSession.CDPEndpoint,
			Viewport:    browserSession.Viewport,
		}
	}

	if session.Status == "completed" || session.Status == "failed" {
		completedAt := session.UpdatedAt
		response.CompletedAt = &completedAt

		duration := completedAt.Sub(session.CreatedAt).Milliseconds()
		response.Duration = &duration

		if resultData, exists := session.Metadata["result"]; exists {
			if resultMap, ok := resultData.(map[string]interface{}); ok {
				response.Result = parseTaskResult(resultMap)
			}
		}
	}

	return response
}

func CreateErrorResponse(errorType, message string, code int, r *http.Request) *EnhancedTaskResponse {
	baseURL := generateBaseURL(r)

	return &EnhancedTaskResponse{
		Success:   false,
		TaskId:    "",
		SessionId: "",
		Status:    "failed",
		Task:      "",
		CreatedAt: time.Now(),
		StreamingUrl: "",
		WebSocketUrl: generateWebSocketURL(baseURL),
		Message:      "Task failed to start",
		Error: &ErrorInfo{
			Type:    errorType,
			Message: message,
			Code:    code,
		},
	}
}

func parseTaskResult(resultMap map[string]interface{}) *TaskResult {
	result := &TaskResult{
		Success:  false,
		Output:   "",
		Steps:    []TaskStep{},
		Metadata: make(map[string]interface{}),
	}

	if success, ok := resultMap["success"].(bool); ok {
		result.Success = success
	}

	if output, ok := resultMap["output"].(string); ok {
		result.Output = output
	} else if extractedContent, ok := resultMap["extracted_content"].(string); ok {
		result.Output = extractedContent
	}

	if tokenUsage, ok := resultMap["token_usage"].(map[string]interface{}); ok {
		result.TokenUsage = parseTokenUsage(tokenUsage)
	}

	for key, value := range resultMap {
		if key != "success" && key != "output" && key != "extracted_content" && key != "token_usage" {
			result.Metadata[key] = value
		}
	}

	return result
}

func parseTokenUsage(tokenMap map[string]interface{}) *TokenUsage {
	usage := &TokenUsage{
		Model: "gpt-4.1",
	}

	if total, ok := tokenMap["total_tokens"].(float64); ok {
		usage.TotalTokens = int(total)
	}
	if prompt, ok := tokenMap["prompt_tokens"].(float64); ok {
		usage.PromptTokens = int(prompt)
	}
	if completion, ok := tokenMap["completion_tokens"].(float64); ok {
		usage.CompletionTokens = int(completion)
	}
	if cost, ok := tokenMap["total_cost"].(float64); ok {
		usage.TotalCost = cost
	}
	if model, ok := tokenMap["model"].(string); ok {
		usage.Model = model
	}

	return usage
}

// URL Generation Helper Functions
func generateBaseURL(r *http.Request) string {
	host := r.Host
	if host == "" {
		cfg := config.GetConfig()
		host = fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	}

	protocol := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		protocol = "https"
	}

	return fmt.Sprintf("%s://%s", protocol, host)
}

func generateAutomationURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/live-automation/%s", baseURL, sessionId)
}

func generateStreamingURL(baseURL, sessionId string) string {
	return fmt.Sprintf("%s/api/stream-screencast/%s", baseURL, sessionId)
}

func generateWebSocketURL(baseURL string) string {
	if strings.HasPrefix(baseURL, "https://") {
		return strings.Replace(baseURL, "https://", "wss://", 1) + "/ws"
	}
	return strings.Replace(baseURL, "http://", "ws://", 1) + "/ws"
}

// Utility functions to reset streaming flags
func ResetAllStreamingFlags() {
	log.Printf("🧹 Resetting all streaming flags on startup...")

	browserManager.mutex.Lock()
	for _, browserSession := range browserManager.sessions {
		browserSession.Streaming = false
	}
	browserManager.mutex.Unlock()

	log.Printf("✅ All streaming flags reset")
}

func StartWebSocketManager() {
	go wsManager.run()
}

func StartHealthMonitoring() {
	browserManager.startHealthMonitoring()
	log.Printf("🏥 Started browser health monitoring")
}
