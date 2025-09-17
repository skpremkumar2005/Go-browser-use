package browser

import (
	"time"
)

// Browser Session DTOs
type CreateBrowserSessionDto struct {
	Viewport ViewportDto `json:"viewport"`
}

type ViewportDto struct {
	Width  int `json:"width" validate:"required,min=100,max=3840"`
	Height int `json:"height" validate:"required,min=100,max=2160"`
}

type BrowserSessionResponseDto struct {
	ID              string                 `json:"id"`
	CDPEndpoint     string                 `json:"cdpEndpoint"`
	CDPPort         int                    `json:"cdpPort"`
	Viewport        ViewportDto            `json:"viewport"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastActivity    time.Time              `json:"lastActivity"`
	Streaming       bool                   `json:"streaming"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	HealthStatus    string                 `json:"healthStatus"`
	AutoRecovery    bool                   `json:"autoRecovery"`
	TaskCompleted   bool                   `json:"taskCompleted"`
	CleanupDuration time.Duration          `json:"cleanupDuration"`
}

// Script Task DTOs
type ScriptTaskRequestDto struct {
	Task     string `json:"task" validate:"required,min=1"`
	MaxSteps int    `json:"maxSteps,omitempty"`
}

type EnhancedTaskResponseDto struct {
	Success       bool                   `json:"success"`
	TaskId        string                 `json:"id"`
	SessionId     string                 `json:"sessionId"`
	Status        string                 `json:"status"`
	Task          string                 `json:"task"`
	CreatedAt     time.Time              `json:"createdAt"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty"`
	Duration      *int64                 `json:"duration,omitempty"`
	AutomationUrl string                 `json:"live_url"`
	StreamingUrl  string                 `json:"streaming_url"`
	WebSocketUrl  string                 `json:"websocket_url"`
	BrowserInfo   *BrowserInfoDto        `json:"browser_info,omitempty"`
	Result        *TaskResultDto         `json:"result,omitempty"`
	Message       string                 `json:"message"`
	Error         *ErrorInfoDto          `json:"error,omitempty"`
}

type BrowserInfoDto struct {
	BrowserId   string      `json:"browserId"`
	CDPEndpoint string      `json:"cdpEndpoint"`
	Viewport    ViewportDto `json:"viewport"`
	CurrentUrl  string      `json:"currentUrl,omitempty"`
}

type TaskResultDto struct {
	Success    bool                   `json:"success"`
	Output     string                 `json:"output"`
	Steps      []TaskStepDto          `json:"steps,omitempty"`
	TokenUsage *TokenUsageDto         `json:"token_usage,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type TaskStepDto struct {
	Step      int       `json:"step"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type TokenUsageDto struct {
	TotalTokens      int     `json:"total_tokens"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalCost        float64 `json:"total_cost"`
	Model            string  `json:"model"`
}

type ErrorInfoDto struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Browser Action DTOs
type BrowserActionDto struct {
	Action string      `json:"action" validate:"required"`
	Data   interface{} `json:"data"`
}

type ClickActionDto struct {
	X int `json:"x" validate:"required"`
	Y int `json:"y" validate:"required"`
}

type TypeActionDto struct {
	Text string `json:"text" validate:"required"`
}

type KeyActionDto struct {
	Key string `json:"key" validate:"required"`
}

type ScrollActionDto struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type NavigateActionDto struct {
	URL string `json:"url" validate:"required,url"`
}

// System Status DTOs
type SystemStatusDto struct {
	Status           string                 `json:"status"`
	Timestamp        time.Time              `json:"timestamp"`
	ActiveSessions   int                    `json:"activeSessions"`
	MaxSessions      int                    `json:"maxSessions"`
	BrowserSessions  []BrowserSessionDto    `json:"browserSessions"`
	ScriptSessions   []ScriptSessionDto     `json:"scriptSessions"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type BrowserSessionDto struct {
	ID           string    `json:"id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActivity time.Time `json:"lastActivity"`
	CDPPort      int       `json:"cdpPort"`
}

type ScriptSessionDto struct {
	ID        string    `json:"id"`
	Task      string    `json:"task"`
	Status    string    `json:"status"`
	BrowserID string    `json:"browserId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WebSocket Message DTOs
type WebSocketMessageDto struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type StreamingDataDto struct {
	SessionID string `json:"sessionId"`
	Image     string `json:"image"` // base64 encoded
	Timestamp time.Time `json:"timestamp"`
}

type TaskUpdateDto struct {
	TaskID    string    `json:"taskId"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Step      int       `json:"step,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ScriptTaskWithBrowserDto for creating task with existing browser (old code architecture)
type ScriptTaskWithBrowserDto struct {
	Task        string `json:"task" validate:"required"`
	MaxSteps    int    `json:"maxSteps"`
	BrowserID   string `json:"browserId" validate:"required"`
	CDPEndpoint string `json:"cdpEndpoint" validate:"required"`
}

// ScriptTaskWithLLMDto for creating task with existing browser and LLM configuration
type ScriptTaskWithLLMDto struct {
	Task        string      `json:"task" validate:"required"`
	MaxSteps    int         `json:"maxSteps"`
	BrowserID   string      `json:"browserId" validate:"required"`
	CDPEndpoint string      `json:"cdpEndpoint" validate:"required"`
	LLMConfig   LLMModelDto `json:"llmConfig" validate:"required"`
}

// LLM Model Configuration DTOs
type LLMModelDto struct {
	ApiKey     string `json:"apiKey" validate:"required"`
	Provider   string `json:"provider" validate:"required"`
	Endpoint   string `json:"endpoint" validate:"required"`
	Deployment string `json:"deployment" validate:"required"`
	LLMModel   string `json:"llmModel" validate:"required"`
}

// ExecuteTaskAndStream Request DTO
type ExecuteTaskRequestDto struct {
	MaxSteps int         `json:"maxSteps" validate:"min=1,max=100"`
	Task     string      `json:"task" validate:"required,min=1"`
	LLMModel LLMModelDto `json:"llm_model" validate:"required"`
}

// ExecuteTaskAndStream Response DTO 
type ExecuteTaskResponseDto struct {
	Success       bool   `json:"success"`
	ID            string `json:"id"`
	SessionID     string `json:"sessionId"`
	SessionReused bool   `json:"session_reused"`
	LiveURL       string `json:"live_url"`
	SocketURL     string `json:"socket_url"`
}