#!/usr/bin/env python3
"""
Browser Task Script with proper browser-use integration
Based on unified-browser-platform architecture but adapted for Go backend
"""

import asyncio
import json
import logging
import os
import sys
import time
from typing import Optional, Dict, Any
import threading
import queue

# Configure logging to capture browser-use logs
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Custom log handler to capture agent logs
class AgentLogHandler(logging.Handler):
    def __init__(self):
        super().__init__()
        self.logs = []
        self.steps = []
        self.step_counter = 0
        
    def sanitize_message(self, message):
        """Remove or replace Unicode characters that cause encoding issues"""
        # Replace common emojis with text equivalents
        emoji_replacements = {
            '🚀': '[START]',
            '✅': '[SUCCESS]',
            '❌': '❌[ERROR]',
            '🎯': '[TASK]',
            '📍': '[STEP]',
            '🦾': '[ACTION]',
            '👍': '[GOOD]',
            '🔗': '[LINK]',
            '📄': '[CONTENT]',
            '⌨️': '[INPUT]',
            '🔍': '[SEARCH]',
            '📊': '[DATA]',
            '🌐': '[WEB]',
            '👉': '->',
            '📋': '[INFO]',
            '🤖': '[BOT]',
            '⚡': '[FAST]',
            '💡': '[TIP]',
            '🔧': '[CONFIG]',
            '📸': '[SCREENSHOT]',
            '🎬': '[RECORDING]',
            '🛡️': '[SECURE]',
            '🥷': '[STEALTH]'
        }
        
        # Replace emojis
        for emoji, replacement in emoji_replacements.items():
            message = message.replace(emoji, replacement)
        
        # Remove ANSI color codes
        import re
        ansi_escape = re.compile(r'\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])')
        message = ansi_escape.sub('', message)
        
        # Ensure the message is encodable
        try:
            message.encode('ascii')
            return message
        except UnicodeEncodeError:
            # Replace any remaining problematic characters
            return message.encode('ascii', errors='replace').decode('ascii')
        
    def emit(self, record):
        try:
            raw_message = record.getMessage()
            sanitized_message = self.sanitize_message(raw_message)
            
            log_entry = {
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime(record.created)),
                "level": record.levelname.lower(),
                "type": "stdout",
                "message": sanitized_message,
                "step": None,
                "action": None
            }
            
            # Detect different types of messages and create user-friendly versions
            message = record.getMessage()
            
            # Detect steps
            if "Step" in message and ":" in message:
                self.step_counter += 1
                log_entry["type"] = "step"
                log_entry["message"] = f"Starting Step {self.step_counter}"
                step_info = {
                    "id": f"step-{self.step_counter}",
                    "step": self.step_counter,
                    "evaluation_previous_goal": "",
                    "next_goal": f"Step {self.step_counter}: Processing task",
                    "url": ""
                }
                self.steps.append(step_info)
                
            elif "Eval:" in message:
                log_entry["type"] = "evaluation" 
                if "Success" in message:
                    log_entry["message"] = "Previous action completed successfully"
                    # Update last step's evaluation
                    if self.steps and len(self.steps) > 1:
                        self.steps[-2]["evaluation_previous_goal"] = "Previous action completed successfully"
                else:
                    log_entry["message"] = "Evaluating previous action"
                    if self.steps and len(self.steps) > 1:
                        self.steps[-2]["evaluation_previous_goal"] = "Evaluating previous action"
                        
            elif "ACTION" in message or "action" in message.lower():
                log_entry["type"] = "action"
                if "go_to_url" in message:
                    log_entry["message"] = "Navigating to webpage"
                elif "click" in message.lower():
                    log_entry["message"] = "Clicking on element"
                elif "input_text" in message:
                    log_entry["message"] = "Typing text into input field"
                elif "send_keys" in message:
                    log_entry["message"] = "Pressing keyboard keys"
                elif "extract_structured_data" in message:
                    log_entry["message"] = "Analyzing page content"
                elif "done" in message:
                    log_entry["message"] = "Task completed"
                else:
                    log_entry["message"] = "Performing browser action"
                    
            elif "Next goal:" in message:
                log_entry["type"] = "goal"
                goal_text = message.split("Next goal:")[-1].strip()
                log_entry["message"] = f"Planning: {goal_text[:100]}..."
                # Update last step's next goal
                if self.steps:
                    self.steps[-1]["next_goal"] = goal_text[:200]
                    
            elif "Navigated to" in message:
                log_entry["type"] = "navigation"
                log_entry["message"] = "Successfully navigated to webpage"
                
            elif "Typed" in message:
                log_entry["type"] = "input"
                log_entry["message"] = "Text input completed"
                
            elif "Sent keys" in message:
                log_entry["type"] = "keyboard" 
                log_entry["message"] = "Keyboard action completed"
                
            elif "Result:" in message:
                log_entry["type"] = "task_completion"
                log_entry["message"] = "Task completed with results"
                
            elif "completed successfully" in message.lower():
                log_entry["type"] = "task_completion"
                log_entry["message"] = "[SUCCESS] Task completed successfully"
                
            elif "error" in message.lower() or "failed" in message.lower():
                log_entry["type"] = "error"
                log_entry["level"] = "error"
                log_entry["message"] = f"Error occurred: {sanitized_message[:100]}..."
                
            elif "HTTP Request" in message:
                log_entry["type"] = "network"
                log_entry["message"] = "Making API request"
                
            elif "Connecting to" in message:
                log_entry["type"] = "connection"
                log_entry["message"] = "Connecting to browser"
                
            elif "Starting task:" in message:
                log_entry["type"] = "task_start" 
                task_desc = message.split(":", 1)[1].strip() if ":" in message else message
                log_entry["message"] = f"[TASK] Starting task: {task_desc}"
                
            elif "browser-use version" in message:
                log_entry["type"] = "startup"
                log_entry["message"] = "Browser automation system started"
                
            # Clean up stdout messages - only keep meaningful ones
            elif log_entry["type"] == "stdout":
                if sanitized_message.strip() in ["", "\n"] or len(sanitized_message.strip()) < 3:
                    return  # Skip empty or very short messages
                elif any(skip in sanitized_message.lower() for skip in ["2025-", "debug", "trace"]):
                    return  # Skip timestamp-only or debug messages
                else:
                    log_entry["message"] = sanitized_message.strip()[:200]  # Truncate long messages
            
            self.logs.append(log_entry)
            
            # Print log for real-time monitoring
            print(f"[LOG] {log_entry['type'].upper()}: {log_entry['message']}", flush=True)
            
        except Exception as e:
            print(f"[LOG_ERROR] Failed to process log: {e}", flush=True)

# Global log handler
log_handler = AgentLogHandler()

try:
    from browser_use import Agent
    from browser_use.browser import BrowserSession, BrowserProfile
    from browser_use.llm import ChatAzureOpenAI
except ImportError as e:
    logger.error(f"❌ Failed to import browser_use: {e}")
    logger.error("Please install browser_use: pip install browser_use")
    sys.exit(1)

class UnifiedBrowserUseAgent:
    """
    Unified Browser Use Agent that connects to existing browser sessions
    Based on unified-browser-platform architecture but adapted for Go backend
    """
    
    def __init__(self):
        self.llm = None
        self.agent = None
        self.load_environment()
    
    def load_environment(self):
        """Load and validate environment variables"""
        try:
            # Setup logging to capture browser-use logs
            browser_use_logger = logging.getLogger('browser_use')
            browser_use_logger.setLevel(logging.INFO)
            browser_use_logger.addHandler(log_handler)
            
            # Also capture root logger for general messages
            root_logger = logging.getLogger()
            root_logger.addHandler(log_handler)
            
            # Azure OpenAI configuration
            api_key = os.getenv("AZURE_OPENAI_API_KEY")
            api_base = os.getenv("AZURE_OPENAI_API_BASE") or os.getenv("AZURE_OPENAI_ENDPOINT")
            api_version = os.getenv("AZURE_OPENAI_API_VERSION", "2024-02-15-preview")
            deployment_name = os.getenv("AZURE_OPENAI_DEPLOYMENT_NAME")
            
            if not all([api_key, api_base, deployment_name]):
                logger.error("❌ Missing required Azure OpenAI environment variables")
                logger.error("Required: AZURE_OPENAI_API_KEY, AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT_NAME")
                sys.exit(1)
            
            # Import ChatAzureOpenAI for proper Azure OpenAI configuration
            from browser_use.llm import ChatAzureOpenAI
            
            # Create ChatAzureOpenAI for Azure OpenAI (correct approach from browser_use_agent.py)
            # Configure Azure OpenAI LLM
            self.llm = ChatAzureOpenAI(
                model=deployment_name,
                api_key=api_key,
                azure_endpoint=api_base,
                azure_deployment=deployment_name,
                api_version=api_version,
                temperature=0.7
            )
            
            logger.info("✅ Azure OpenAI LLM configured successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to configure LLM: {e}")
            sys.exit(1)
    
    async def create_agent(self, task: str, cdp_endpoint: str | None = None, max_steps: int = 10):
        """
        Create browser-use agent with enhanced stealth mode
        """
        try:
            import random
            
            browser_session = None
            
            # Create enhanced stealth browser profile
            browser_profile = BrowserProfile(
                # Core stealth settings
                stealth=True,
                disable_security=False,
                headless=False,  # Set to False for live streaming
                
                # Randomized human-like timings (more realistic)
                wait_between_actions=random.uniform(1.0, 2.0),
                minimum_wait_page_load_time=random.uniform(0.5, 1.0),
                wait_for_network_idle_page_load_time=random.uniform(1.0, 2.0),
                maximum_wait_page_load_time=random.uniform(10.0, 15.0),
                
                # Disable automation hints
                highlight_elements=False,
            )
            
            # Check if we should connect to existing browser
            if cdp_endpoint and cdp_endpoint.startswith('ws://'):
                # Connect to existing browser via WebSocket CDP endpoint
                logger.info(f"🔗 Connecting to existing browser session with enhanced stealth: {cdp_endpoint}")
                
                try:
                    browser_session = BrowserSession(
                        cdp_url=cdp_endpoint,
                        is_local=True,
                        browser_profile=browser_profile
                    )
                    
                    logger.info(f"✅ Created ultra-stealth browser session for existing browser")
                    
                except Exception as e:
                    logger.error(f"❌ Failed to create stealth browser session: {e}")
                    raise
            else:
                # Create new browser session with enhanced stealth
                logger.info(f"🆕 Creating new ultra-stealth browser instance")
                browser_session = BrowserSession(browser_profile=browser_profile)
            
            # Add human-like startup delay
            startup_delay = random.uniform(1.0, 3.0)
            logger.info(f"⏳ Adding human-like startup delay: {startup_delay:.1f}s")
            await asyncio.sleep(startup_delay)
            
            # Create agent with enhanced settings
            self.agent = Agent(
                task=task,
                llm=self.llm,
                max_steps=max_steps + 3,  # Give more steps for natural behavior
                browser_session=browser_session,
                use_vision=False,
                calculate_cost=True,
            )
            
            # Get the agent's logger to capture logs
            agent_logger = logging.getLogger('browser_use.agent')
            agent_logger.setLevel(logging.INFO)
            agent_logger.addHandler(log_handler)
            
            # Also capture service logger
            service_logger = logging.getLogger('browser_use.agent.service')
            service_logger.setLevel(logging.INFO)
            service_logger.addHandler(log_handler)
            
            logger.info("✅ Ultra-stealth browser-use agent created successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to create ultra-stealth agent: {e}")
            raise
    
    async def run_task(self, task: str, cdp_endpoint: str | None = None, max_steps: int = 10) -> Dict[str, Any]:
        """
        Run the browser automation task using browser-use
        Based on unified-browser-platform architecture
        """
        start_time = time.time()
        
        try:
            # Create agent
            await self.create_agent(task, cdp_endpoint, max_steps)
            
            # Clear previous logs and steps
            log_handler.logs.clear()
            log_handler.steps.clear()
            log_handler.step_counter = 0
            
            # Add a step tracking interceptor for the agent
            original_run = self.agent.run
            
            async def intercepted_run():
                """Intercepted run method to capture more detailed steps"""
                logger.info(f"🎯 Starting task: {task}")
                log_handler.logs.append({
                    "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime()),
                    "level": "info",
                    "type": "task_start",
                    "message": f"Starting task: {task}",
                    "step": None,
                    "action": None
                })
                
                try:
                    result = await original_run()
                    logger.info("✅ Task completed successfully")
                    log_handler.logs.append({
                        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime()),
                        "level": "info",
                        "type": "task_completion",
                        "message": f"Task completed successfully: {result}",
                        "step": None,
                        "action": None
                    })
                    return result
                except Exception as e:
                    logger.error(f"❌ Task failed: {e}")
                    log_handler.logs.append({
                        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S.000Z", time.gmtime()),
                        "level": "error", 
                        "type": "error",
                        "message": f"Task failed: {str(e)}",
                        "step": None,
                        "action": None
                    })
                    raise
            
            # Execute the task with interception
            result = await intercepted_run()
            
            end_time = time.time()
            duration_ms = int((end_time - start_time) * 1000)
            
            # Extract task summary from the final result
            task_summary = ""
            if result and hasattr(result, 'history') and result.history:
                # Get the last action result which contains the final extracted content
                last_result = result.history[-1]
                if hasattr(last_result, 'result') and last_result.result:
                    final_action = last_result.result[0] if isinstance(last_result.result, list) and last_result.result else last_result.result
                    if hasattr(final_action, 'extracted_content') and final_action.extracted_content:
                        # Clean up the extracted content to use as summary
                        task_summary = final_action.extracted_content.strip()
                        if len(task_summary) > 500:  # Truncate if too long
                            task_summary = task_summary[:500] + "..."
            
            # Clean up step messages - remove ANSI codes
            cleaned_steps = []
            for step in log_handler.steps:
                cleaned_step = step.copy()
                if cleaned_step.get("next_goal"):
                    # Remove ANSI escape codes from next_goal
                    import re
                    ansi_escape = re.compile(r'\x1b\[[0-9;]*[mK]')
                    cleaned_step["next_goal"] = ansi_escape.sub('', cleaned_step["next_goal"]).strip()
                cleaned_steps.append(cleaned_step)
            
            # Print final logs as JSON for Go to parse
            final_logs_data = {
                "logs": log_handler.logs,
                "steps": cleaned_steps,
                "logsSummary": {
                    "totalActions": len([l for l in log_handler.logs if l["type"] in ["action", "click", "input"]]),
                    "browserActions": len([l for l in log_handler.logs if l["type"] == "action"]),
                    "steps": len(cleaned_steps),
                    "errors": len([l for l in log_handler.logs if l["level"] == "error"])
                },
                "duration": duration_ms,
                "durationHuman": f"{duration_ms//60000}m {(duration_ms%60000)//1000}s" if duration_ms > 60000 else f"{duration_ms//1000}s",
                "summary": task_summary
            }
            
            # Output logs data as a separate JSON line for Go to parse
            print(f"[LOGS_DATA] {json.dumps(final_logs_data)}", flush=True)
            
            # Return result in the same format as Node.js version
            return {
                "success": True,
                "result": str(result) if result else "Task completed",
                "cdp_endpoint": cdp_endpoint,
                "executed_at": time.time(),
                "logs_captured": len(log_handler.logs),
                "steps_captured": len(cleaned_steps),
                "summary": task_summary
            }
            
        except Exception as e:
            end_time = time.time()
            duration_ms = int((end_time - start_time) * 1000)
            
            logger.error(f"❌ Task execution failed: {e}")
            
            # Still output logs even on failure
            final_logs_data = {
                "logs": log_handler.logs,
                "steps": log_handler.steps,
                "logsSummary": {
                    "totalActions": len([l for l in log_handler.logs if l["type"] in ["action", "click", "input"]]),
                    "browserActions": len([l for l in log_handler.logs if l["type"] == "action"]),
                    "steps": len(log_handler.steps),
                    "errors": len([l for l in log_handler.logs if l["level"] == "error"]) + 1  # +1 for this error
                },
                "duration": duration_ms,
                "durationHuman": f"{duration_ms//60000}m {(duration_ms%60000)//1000}s" if duration_ms > 60000 else f"{duration_ms//1000}s"
            }
            
            print(f"[LOGS_DATA] {json.dumps(final_logs_data)}", flush=True)
            
            # Ensure error is JSON serializable
            error_msg = str(e)
            if not error_msg:
                error_msg = "Unknown error occurred"
            
            return {
                "success": False,
                "error": error_msg,
                "error_type": type(e).__name__,
                "cdp_endpoint": cdp_endpoint,
                "executed_at": time.time(),
                "logs_captured": len(log_handler.logs),
                "steps_captured": len(log_handler.steps)
            }

async def main():
    """Main script entry point"""
    if len(sys.argv) < 4:
        print("Usage: python browser_task_fixed.py <session_id> <task> <max_steps> [cdp_endpoint]")
        sys.exit(1)
    # task="search in bing"
    session_id = sys.argv[1]
    task = sys.argv[2]
    print(type(task))
    max_steps = int(sys.argv[3])
    cdp_endpoint = sys.argv[4] if len(sys.argv) > 4 else None
    task=task.replace("google", "bing")
    
      # Handle newlines in task
    logger.info(f"🚀 Starting browser task script")
    logger.info(f"📋 Session ID: {session_id}")
    logger.info(f"🎯 Task: {task}")
    logger.info(f"📊 Max Steps: {max_steps}")
    logger.info(f"🔌 CDP Endpoint: {cdp_endpoint}")
    
    # Create and run agent
    agent = UnifiedBrowserUseAgent()
    result = await agent.run_task(task, cdp_endpoint, max_steps)
    
    # Output result to stdout for Go to read
    print(json.dumps(result))
    
    if result["success"]:
        logger.info(f"✅ Task executed successfully")
        sys.exit(0)
    else:
        logger.error(f"❌ Task execution failed")
        sys.exit(1)

if __name__ == "__main__":
    import os
    asyncio.run(main())