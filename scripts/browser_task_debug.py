#!/usr/bin/env python3
"""
Debug Browser Task Script - to identify where the original script hangs
"""

import asyncio
import json
import logging
import os
import sys
import time
from typing import Optional, Dict, Any

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def log_step(step, message):
    """Log each step with timestamp"""
    timestamp = time.strftime("%H:%M:%S")
    print(f"[{timestamp}] STEP {step}: {message}", flush=True)
    logger.info(f"STEP {step}: {message}")

async def main():
    """Main script entry point with debug steps"""
    log_step(1, "Script started")
    
    if len(sys.argv) < 4:
        log_step("ERROR", "Not enough arguments")
        print("Usage: python browser_task_debug.py <session_id> <task> <max_steps> [cdp_endpoint]")
        sys.exit(1)
    
    session_id = sys.argv[1]
    task = sys.argv[2]
    max_steps = int(sys.argv[3])
    cdp_endpoint = sys.argv[4] if len(sys.argv) > 4 else None
    
    log_step(2, f"Arguments parsed - Session: {session_id}, Task: {task}")
    
    # Test environment
    log_step(3, "Checking environment variables")
    api_key = os.getenv("AZURE_OPENAI_API_KEY")
    api_base = os.getenv("AZURE_OPENAI_ENDPOINT")
    deployment = os.getenv("AZURE_OPENAI_DEPLOYMENT_NAME")
    
    if not all([api_key, api_base, deployment]):
        log_step("ERROR", "Missing Azure OpenAI environment variables")
        result = {
            "success": False,
            "error": "Missing Azure OpenAI configuration",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    log_step(4, "Environment variables OK")
    
    # Test imports
    log_step(5, "Testing imports")
    try:
        from browser_use import Agent
        log_step(6, "browser_use.Agent imported")
        
        from browser_use.browser import BrowserSession, BrowserProfile
        log_step(7, "browser_use.browser imports OK")
        
        from browser_use.llm import ChatAzureOpenAI
        log_step(8, "browser_use.llm.ChatAzureOpenAI imported")
        
    except ImportError as e:
        log_step("ERROR", f"Import failed: {e}")
        result = {
            "success": False,
            "error": f"Import error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    # Test LLM creation
    log_step(9, "Creating Azure OpenAI LLM")
    try:
        llm = ChatAzureOpenAI(
            model=deployment,
            api_key=api_key,
            azure_endpoint=api_base,
            azure_deployment=deployment,
            api_version="2024-02-15-preview",
            temperature=0.7
        )
        log_step(10, "LLM created successfully")
    except Exception as e:
        log_step("ERROR", f"LLM creation failed: {e}")
        result = {
            "success": False,
            "error": f"LLM creation error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    # Test browser profile creation
    log_step(11, "Creating browser profile")
    try:
        browser_profile = BrowserProfile(
            stealth=True,
            headless=True,
            wait_between_actions=1.0,
            highlight_elements=False
        )
        log_step(12, "Browser profile created")
    except Exception as e:
        log_step("ERROR", f"Browser profile creation failed: {e}")
        result = {
            "success": False,
            "error": f"Browser profile error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    # Test browser session creation (this is likely where it hangs)
    log_step(13, "Creating browser session - THIS MAY HANG")
    try:
        browser_session = BrowserSession(browser_profile=browser_profile)
        log_step(14, "Browser session created!")
    except Exception as e:
        log_step("ERROR", f"Browser session creation failed: {e}")
        result = {
            "success": False,
            "error": f"Browser session error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    # Test agent creation
    log_step(15, "Creating agent")
    try:
        agent = Agent(
            task=task,
            llm=llm,
            max_steps=max_steps,
            browser_session=browser_session,
            use_vision=False,
            save_conversation_path=None,
            calculate_cost=True,
        )
        log_step(16, "Agent created successfully")
    except Exception as e:
        log_step("ERROR", f"Agent creation failed: {e}")
        result = {
            "success": False,
            "error": f"Agent creation error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)
    
    # Test simple task execution
    log_step(17, "Starting simple task execution")
    try:
        # Just return success without actually running the task for now
        log_step(18, "Task completed (debug mode)")
        
        result = {
            "success": True,
            "result": "Debug task completed - browser session creation worked!",
            "cdp_endpoint": "ws://localhost:9222/devtools/browser/debug",  # dummy
            "executed_at": time.time()
        }
        
        print(json.dumps(result))
        log_step(19, "Result output completed")
        
    except Exception as e:
        log_step("ERROR", f"Task execution failed: {e}")
        result = {
            "success": False,
            "error": f"Task execution error: {e}",
            "executed_at": time.time()
        }
        print(json.dumps(result))
        sys.exit(1)

if __name__ == "__main__":
    log_step(0, "Starting debug script")
    asyncio.run(main())
    log_step(20, "Script completed successfully")