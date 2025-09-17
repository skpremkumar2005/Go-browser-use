package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	cmd.Env = os.Environ()

	log.Printf("🚀 Starting Python script execution...")

	output, err := cmd.CombinedOutput()
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
	} else {
		session.Status = "failed"
		if errorMsg, exists := result["error"]; exists {
			session.Metadata["error"] = errorMsg
		}
	}
	session.UpdatedAt = time.Now()
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
