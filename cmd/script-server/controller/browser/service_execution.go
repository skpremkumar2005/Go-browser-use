package browser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Task execution functions

func ExecuteScriptTask(sessionID, task string, maxSteps int) {
	scriptSessionManager.mutex.RLock()
	session := scriptSessionManager.sessions[sessionID]
	scriptSessionManager.mutex.RUnlock()

	if session == nil {
		log.Printf("❌ Session not found: %s", sessionID)
		return
	}

	scriptSessionManager.mutex.Lock()
	if session.Status != "pending" && session.Status != "queued" {
		scriptSessionManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already in state: %s", sessionID, session.Status)
		return
	}

	if session.Status == "running" {
		scriptSessionManager.mutex.Unlock()
		log.Printf("⚠️ Session %s already running, skipping duplicate execution", sessionID)
		return
	}

	session.Status = "running"
	session.UpdatedAt = time.Now()
	scriptSessionManager.mutex.Unlock()

	browserManager.cancelCleanupTimer(session.BrowserID)

	log.Printf("🎯 Processing task for session %s: %s", sessionID, task)

	browserSession := browserManager.GetSession(session.BrowserID)
	if browserSession == nil {
		log.Printf("❌ Browser session not found: %s", session.BrowserID)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	log.Printf("🔍 Testing CDP endpoint before starting automation: %s", browserSession.CDPEndpoint)
	cdpClient := NewCDPClient(browserSession.CDPEndpoint)

	connectDone := make(chan error, 1)
	go func() {
		connectDone <- cdpClient.Connect()
	}()

	select {
	case err := <-connectDone:
		if err != nil {
			log.Printf("❌ Failed to connect to CDP endpoint: %v", err)
			scriptSessionManager.mutex.Lock()
			session.Status = "failed"
			session.UpdatedAt = time.Now()
			scriptSessionManager.mutex.Unlock()
			return
		}
		cdpClient.Close()
		log.Printf("✅ CDP endpoint validated, browser is ready for automation")
	case <-time.After(10 * time.Second):
		log.Printf("❌ CDP connection timeout after 10 seconds")
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	scriptDir := cfg.Script.ScriptDir
	scriptPath := filepath.Join(scriptDir, cfg.Script.TaskScript)

	log.Printf("🐍 Executing Python script: %s", scriptPath)
	log.Printf("🔌 CDP Endpoint: %s", browserSession.CDPEndpoint)
	log.Printf("🎯 Task: %s", task)
	log.Printf("📊 Max Steps: %d", maxSteps)

	args := []string{
		scriptPath,
		sessionID,
		task,
		strconv.Itoa(maxSteps),
		browserSession.CDPEndpoint,
	}

	cmd := exec.Command(cfg.Script.PythonCommand, args...)
	
	// Set up environment with LLM configuration if provided
	env := os.Environ()
	
	// Get LLM config from session metadata
	if llmConfig, exists := session.Metadata["llm_config"]; exists {
		if config, ok := llmConfig.(*LLMModel); ok {
			log.Printf("🤖 Using provided LLM configuration:")
			log.Printf("   Provider: %s", config.Provider)
			log.Printf("   Model: %s", config.LLMModel)
			log.Printf("   Endpoint: %s", config.Endpoint)
			log.Printf("   Deployment: %s", config.Deployment)
			if config.Version != "" {
				log.Printf("   API Version: %s", config.Version)
			}
			log.Printf("   API Key: %s...%s", config.APIKey[:8], config.APIKey[len(config.APIKey)-8:])
			
			// Set environment variables for the Python script
			env = append(env, fmt.Sprintf("OPENAI_API_KEY=%s", config.APIKey))
			env = append(env, fmt.Sprintf("AZURE_OPENAI_API_KEY=%s", config.APIKey))
			env = append(env, fmt.Sprintf("AZURE_OPENAI_ENDPOINT=%s", config.Endpoint))
			env = append(env, fmt.Sprintf("AZURE_OPENAI_DEPLOYMENT_NAME=%s", config.Deployment))
			
			// Use the version from request, or default to the newer version that supports json_schema
			apiVersion := "2024-08-01-preview"
			if config.Version != "" {
				apiVersion = config.Version
			}
			env = append(env, fmt.Sprintf("AZURE_OPENAI_API_VERSION=%s", apiVersion))
			
			env = append(env, fmt.Sprintf("LLM_PROVIDER=%s", config.Provider))
			env = append(env, fmt.Sprintf("LLM_MODEL=%s", config.LLMModel))
			
			log.Printf("✅ LLM environment variables set successfully")
		} else {
			log.Printf("⚠️ Invalid LLM config format in session metadata")
		}
	} else {
		log.Printf("🔧 Using default LLM configuration from environment")
	}
	
	cmd.Env = env

	log.Printf("🚀 Starting Python script execution...")

	// Create pipes for streaming output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdout pipe: %v", err)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}
	
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("❌ Failed to create stderr pipe: %v", err)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start Python script: %v", err)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	var outputBuffer strings.Builder
	var wg sync.WaitGroup
	
	// Read stdout in real-time
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuffer.WriteString(line + "\n")
			
			// Parse and store logs in real-time
			if strings.Contains(line, "[LOG]") || strings.Contains(line, "[LOGS_DATA]") {
				// Update session with latest logs
				scriptSessionManager.mutex.Lock()
				if session, exists := scriptSessionManager.sessions[sessionID]; exists {
					logs := parseOutputLogs(outputBuffer.String(), sessionID)
					steps := extractStepsFromLogs(outputBuffer.String(), sessionID) 
					
					session.Logs = logs
					session.Steps = steps
					session.UpdatedAt = time.Now()
					scriptSessionManager.sessions[sessionID] = session
				}
				scriptSessionManager.mutex.Unlock()
			}
		}
	}()
	
	// Read stderr in real-time  
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuffer.WriteString("[STDERR] " + line + "\n")
		}
	}()

	// Wait for the command to finish
	err = cmd.Wait()
	wg.Wait() // Wait for all readers to finish

	output := outputBuffer.String()
	if err != nil {
		log.Printf("❌ Script execution failed: %v", err)
		log.Printf("❌ Script output: %s", string(output))
		log.Printf("❌ CDP endpoint used: %s", browserSession.CDPEndpoint)
		scriptSessionManager.mutex.Lock()
		session.Status = "failed"
		session.Metadata["error"] = fmt.Sprintf("Script failed: %v\nOutput: %s", err, string(output))
		session.UpdatedAt = time.Now()
		scriptSessionManager.mutex.Unlock()
		return
	}

	var result map[string]interface{}
	outputStr := string(output)
	log.Printf("📄 Python script output: %s", outputStr)

	// Parse logs and extract steps
	session.Logs = parseOutputLogs(outputStr, sessionID)
	session.Steps = extractStepsFromLogs(outputStr, sessionID)

	lines := strings.Split(outputStr, "\n")
	jsonFound := false

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				jsonFound = true
				log.Printf("✅ Parsed JSON result from line %d: %+v", i, result)
				break
			}
		}
	}

	if !jsonFound {
		jsonStart := strings.LastIndex(outputStr, "{")
		if jsonStart != -1 {
			braceCount := 0
			jsonEnd := -1
			for i := jsonStart; i < len(outputStr); i++ {
				if outputStr[i] == '{' {
					braceCount++
				} else if outputStr[i] == '}' {
					braceCount--
					if braceCount == 0 {
						jsonEnd = i + 1
						break
					}
				}
			}

			if jsonEnd > jsonStart {
				jsonStr := outputStr[jsonStart:jsonEnd]
				if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
					jsonFound = true
					log.Printf("✅ Parsed JSON result from multi-line: %+v", result)
				}
			}
		}
	}

	if !jsonFound {
		log.Printf("⚠️ No valid JSON found in output, creating default result")
		if strings.Contains(outputStr, "✅ Task completed successfully") ||
			strings.Contains(outputStr, "Task completed: True") {
			log.Printf("✅ Task appears to have completed successfully based on logs")
			result = map[string]interface{}{
				"success":    true,
				"result":     "Task completed successfully",
				"raw_output": outputStr,
			}
		} else {
			result = map[string]interface{}{
				"success":    false,
				"error":      "No valid JSON output from script",
				"raw_output": outputStr,
			}
		}
	}

	scriptSessionManager.mutex.Lock()
	success, _ := result["success"].(bool)
	if success {
		session.Status = "completed"
		session.Metadata["result"] = result
		// Extract additional data from result
		if tokenUsageData, exists := result["token_usage"]; exists {
			session.TokenUsage = parseTokenUsageFromResult(tokenUsageData)
		}
		if summaryData, exists := result["summary"]; exists {
			if summaryStr, ok := summaryData.(string); ok {
				session.Summary = summaryStr
			}
		}
		if outputData, exists := result["output"]; exists {
			if outputStr, ok := outputData.(string); ok {
				session.Output = outputStr
			}
		}
	} else {
		session.Status = "failed"
		if errorMsg, exists := result["error"]; exists {
			session.Metadata["error"] = errorMsg
		}
		session.Output = "Task execution failed"
	}
	
	// Set finished time
	now := time.Now()
	session.FinishedAt = &now
	session.UpdatedAt = now
	scriptSessionManager.mutex.Unlock()

	browserSession = browserManager.GetSession(session.BrowserID)
	if browserSession != nil {
		browserSession.mutex.Lock()
		browserSession.TaskCompleted = true
		browserSession.LastUserInteraction = time.Now()
		browserSession.mutex.Unlock()

		browserManager.startCleanupTimer(session.BrowserID, 30*time.Second)
		log.Printf("⏰ Task completed for session %s, cleanup timer started (30s)", sessionID)
	}

	wsManager.broadcastTaskUpdate(sessionID, session.Status)

	log.Printf("✅ Task completed for session %s with status: %s", sessionID, session.Status)
}

// Browser Manager helper functions

func (bm *BrowserManager) startCleanupTimer(sessionID string, duration time.Duration) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return
	}

	if session.CleanupTimer != nil {
		session.CleanupTimer.Stop()
	}

	session.CleanupTimer = time.AfterFunc(duration, func() {
		log.Printf("⏰ Auto-cleanup timer expired for session %s, cleaning up...", sessionID)
		if err := bm.CloseSession(sessionID); err != nil {
			log.Printf("❌ Failed to auto-cleanup session %s: %v", sessionID, err)
		}
	})

	session.CleanupDuration = duration
	bm.mutex.Unlock()

	log.Printf("⏰ Started cleanup timer for session %s (duration: %v)", sessionID, duration)
}

func (bm *BrowserManager) cancelCleanupTimer(sessionID string) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if exists && session.CleanupTimer != nil {
		session.CleanupTimer.Stop()
		session.CleanupTimer = nil
		log.Printf("⏰ Cancelled cleanup timer for session %s", sessionID)
	}
	bm.mutex.Unlock()
}

func (bm *BrowserManager) updateUserInteraction(sessionID string) {
	bm.mutex.Lock()
	session, exists := bm.sessions[sessionID]
	if !exists {
		bm.mutex.Unlock()
		return
	}

	now := time.Now()
	session.LastUserInteraction = now

	if session.TaskCompleted {
		if session.CleanupTimer != nil {
			session.CleanupTimer.Stop()
			session.CleanupTimer = nil
		}
		log.Printf("👤 User interaction detected for session %s, cleanup timer cancelled", sessionID)
	}
	bm.mutex.Unlock()
}

func (bm *BrowserManager) checkSessionHealth(sessionID string) error {
	bm.mutex.RLock()
	session := bm.sessions[sessionID]
	bm.mutex.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	if session.BrowserProcess == nil {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		return fmt.Errorf("browser process is nil")
	}

	if session.BrowserProcess.ProcessState != nil && session.BrowserProcess.ProcessState.Exited() {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		return fmt.Errorf("browser process has exited")
	}

	cdpClient := NewCDPClient(session.CDPEndpoint)
	if err := cdpClient.Connect(); err != nil {
		session.HealthStatus = "unhealthy"
		session.ConnectionErrors++
		session.LastConnectionError = time.Now()
		cdpClient.Close()
		return fmt.Errorf("CDP connection failed: %v", err)
	}
	cdpClient.Close()

	session.HealthStatus = "healthy"
	session.ConnectionErrors = 0
	session.LastHealthCheck = time.Now()
	return nil
}

func (bm *BrowserManager) startHealthMonitoring() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				bm.mutex.RLock()
				sessions := make([]string, 0, len(bm.sessions))
				for sessionID := range bm.sessions {
					sessions = append(sessions, sessionID)
				}
				bm.mutex.RUnlock()

				for _, sessionID := range sessions {
					if err := bm.checkSessionHealth(sessionID); err != nil {
						log.Printf("⚠️ Health check failed for session %s: %v", sessionID, err)

						bm.mutex.RLock()
						session := bm.sessions[sessionID]
						bm.mutex.RUnlock()

						if session != nil && session.AutoRecovery {
							if err := bm.recoverSession(sessionID); err != nil {
								log.Printf("❌ Recovery failed for session %s: %v", sessionID, err)
							}
						}
					}
				}
			case <-time.After(5 * time.Minute):
				bm.cleanupStaleSessions()
			}
		}
	}()
}

func (bm *BrowserManager) recoverSession(sessionID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	session := bm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if !session.AutoRecovery {
		return fmt.Errorf("auto-recovery disabled for session")
	}

	if session.RecoveryAttempts >= session.MaxRecoveryAttempts {
		return fmt.Errorf("max recovery attempts reached (%d)", session.MaxRecoveryAttempts)
	}

	log.Printf("🔄 Attempting to recover session %s (attempt %d/%d)", sessionID, session.RecoveryAttempts+1, session.MaxRecoveryAttempts)

	if session.BrowserProcess != nil {
		session.BrowserProcess.Process.Kill()
		session.BrowserProcess.Wait()
	}

	viewport := session.Viewport
	newSession, err := bm.CreateBrowserSession(sessionID+"_recovered", viewport)
	if err != nil {
		session.RecoveryAttempts++
		return fmt.Errorf("failed to recreate browser session: %v", err)
	}

	session.BrowserProcess = newSession.BrowserProcess
	session.CDPEndpoint = newSession.CDPEndpoint
	session.CDPPort = newSession.CDPPort
	session.Status = "ready"
	session.HealthStatus = "healthy"
	session.ConnectionErrors = 0
	session.RecoveryAttempts++
	session.LastActivity = time.Now()

	log.Printf("✅ Successfully recovered session %s", sessionID)
	return nil
}

func (bm *BrowserManager) cleanupStaleSessions() {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute)
	sessionsToCleanup := make([]string, 0)

	for sessionID, session := range bm.sessions {
		if session.LastActivity.Before(cutoff) && !session.Streaming {
			sessionsToCleanup = append(sessionsToCleanup, sessionID)
		}
	}

	for _, sessionID := range sessionsToCleanup {
		session := bm.sessions[sessionID]
		log.Printf("🧹 Cleaning up stale session: %s", sessionID)

		session.mutex.Lock()
		session.Streaming = false
		session.mutex.Unlock()

		if session.BrowserProcess != nil {
			session.BrowserProcess.Process.Kill()
			session.BrowserProcess.Wait()
		}

		delete(bm.sessions, sessionID)

		bm.sessionMutex.Lock()
		if bm.activeSessions > 0 {
			bm.activeSessions--
		}
		bm.sessionMutex.Unlock()
	}

	if len(sessionsToCleanup) > 0 {
		log.Printf("🧹 Cleaned up %d stale sessions (active sessions: %d/%d)",
			len(sessionsToCleanup), bm.activeSessions, bm.maxConcurrentSessions)
	}
}

// Helper functions for parsing execution results and logs
func parseOutputLogs(output, sessionID string) []LogEntry {
	logs := []LogEntry{}
	lines := strings.Split(output, "\n")
	
	// First, try to find the structured logs data from Python
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[LOGS_DATA] ") {
			jsonStr := strings.TrimPrefix(line, "[LOGS_DATA] ")
			var logsData struct {
				Logs []struct {
					Timestamp string      `json:"timestamp"`
					Level     string      `json:"level"`
					Type      string      `json:"type"`
					Message   string      `json:"message"`
					Step      interface{} `json:"step"`
					Action    interface{} `json:"action"`
				} `json:"logs"`
				Steps []struct {
					ID                     string `json:"id"`
					Step                   int    `json:"step"`
					EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
					NextGoal               string `json:"next_goal"`
					URL                    string `json:"url"`
				} `json:"steps"`
				LogsSummary struct {
					TotalActions    int `json:"totalActions"`
					BrowserActions  int `json:"browserActions"`
					Steps           int `json:"steps"`
					Errors          int `json:"errors"`
				} `json:"logsSummary"`
				Duration      int    `json:"duration"`
				DurationHuman string `json:"durationHuman"`
			}
			
			if err := json.Unmarshal([]byte(jsonStr), &logsData); err == nil {
				log.Printf("✅ Parsed structured logs data: %d logs, %d steps", len(logsData.Logs), len(logsData.Steps))
				
				// Convert to our LogEntry format
				for _, logItem := range logsData.Logs {
					logEntry := LogEntry{
						Timestamp: logItem.Timestamp,
						Level:     logItem.Level,
						Type:      logItem.Type,
						Message:   logItem.Message,
						Step:      logItem.Step,
						Action:    logItem.Action,
					}
					logs = append(logs, logEntry)
				}
				
				return logs
			} else {
				log.Printf("⚠️ Failed to parse structured logs data: %v", err)
			}
		}
	}
	
	// Fallback to parsing regular log lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[LOG]") || strings.HasPrefix(line, "[LOGS_DATA]") {
			continue
		}
		
		// Parse different types of log entries
		logEntry := LogEntry{
			Timestamp: time.Now().Format("2006-01-02T15:04:05.000Z"),
			Level:     "info",
			Type:      "stdout",
			Message:   line,
			Step:      nil,
			Action:    nil,
		}
		
		// Detect different types of log entries
		if strings.Contains(line, "Step") && strings.Contains(line, ":") {
			logEntry.Type = "step"
		} else if strings.Contains(line, "ERROR") || strings.Contains(line, "❌") {
			logEntry.Type = "error"
			logEntry.Level = "error"
		} else if strings.Contains(line, "ACTION") || strings.Contains(line, "click") || strings.Contains(line, "input") {
			logEntry.Type = "action"
		} else if strings.Contains(line, "goal") || strings.Contains(line, "Next goal") {
			logEntry.Type = "goal"
		} else if strings.Contains(line, "Eval") || strings.Contains(line, "evaluation") {
			logEntry.Type = "evaluation"
		} else if strings.Contains(line, "Task completed") {
			logEntry.Type = "task_completion"
		}
		
		logs = append(logs, logEntry)
	}
	
	return logs
}

func extractStepsFromLogs(output, sessionID string) []TaskStepDetail {
	steps := []TaskStepDetail{}
	lines := strings.Split(output, "\n")
	
	// First, try to find the structured steps data from Python
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[LOGS_DATA] ") {
			jsonStr := strings.TrimPrefix(line, "[LOGS_DATA] ")
			var logsData struct {
				Steps []struct {
					ID                     string `json:"id"`
					Step                   int    `json:"step"`
					EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
					NextGoal               string `json:"next_goal"`
					URL                    string `json:"url"`
				} `json:"steps"`
			}
			
			if err := json.Unmarshal([]byte(jsonStr), &logsData); err == nil {
				log.Printf("✅ Parsed structured steps data: %d steps", len(logsData.Steps))
				
				// Convert to our TaskStepDetail format
				for _, stepItem := range logsData.Steps {
					step := TaskStepDetail{
						ID:                     stepItem.ID,
						Step:                   stepItem.Step,
						EvaluationPreviousGoal: stepItem.EvaluationPreviousGoal,
						NextGoal:               stepItem.NextGoal,
						URL:                    stepItem.URL,
					}
					steps = append(steps, step)
				}
				
				return steps
			} else {
				log.Printf("⚠️ Failed to parse structured steps data: %v", err)
			}
		}
	}
	
	// Fallback to parsing from log lines
	stepCounter := 1
	var currentStep *TaskStepDetail
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.Contains(line, "Step") && strings.Contains(line, ":") {
			// Finalize previous step if exists
			if currentStep != nil {
				steps = append(steps, *currentStep)
			}
			
			// Create new step
			currentStep = &TaskStepDetail{
				ID:                     fmt.Sprintf("%s-step-%d", sessionID, stepCounter),
				Step:                   stepCounter,
				EvaluationPreviousGoal: "",
				NextGoal:               extractGoalFromMessage(line),
				URL:                    "",
			}
			stepCounter++
		} else if currentStep != nil {
			if strings.Contains(line, "eval") || strings.Contains(line, "Eval") {
				currentStep.EvaluationPreviousGoal = extractGoalFromMessage(line)
			} else if strings.Contains(line, "goal") || strings.Contains(line, "Goal") {
				currentStep.NextGoal = extractGoalFromMessage(line)
			}
		}
	}
	
	// Add final step if exists
	if currentStep != nil {
		steps = append(steps, *currentStep)
	}
	
	return steps
}

func extractGoalFromMessage(message string) string {
	// Extract goal text from various message formats
	if strings.Contains(message, "Next goal:") {
		parts := strings.SplitN(message, "Next goal:", 2)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(message, "Eval:") {
		parts := strings.SplitN(message, "Eval:", 2)
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(message)
}

func parseTokenUsageFromResult(tokenUsageData interface{}) *TokenUsageDetail {
	if tokenMap, ok := tokenUsageData.(map[string]interface{}); ok {
		tokenUsage := &TokenUsageDetail{
			Model: "gpt-4.1",
		}
		
		if total, ok := tokenMap["total_tokens"].(float64); ok {
			tokenUsage.TotalTokens = int(total)
		}
		if prompt, ok := tokenMap["prompt_tokens"].(float64); ok {
			tokenUsage.PromptTokens = int(prompt)
		}
		if completion, ok := tokenMap["completion_tokens"].(float64); ok {
			tokenUsage.CompletionTokens = int(completion)
		}
		if cost, ok := tokenMap["total_cost"].(float64); ok {
			tokenUsage.TotalCost = cost
		}
		if model, ok := tokenMap["model"].(string); ok {
			tokenUsage.Model = model
		}
		
		return tokenUsage
	}
	return nil
}
