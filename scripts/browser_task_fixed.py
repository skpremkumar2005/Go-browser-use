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

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

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
                deterministic_rendering=False,
                enable_default_extensions=True,
                headless=True,
                
                # Randomized human-like timings (more realistic)
                wait_between_actions=random.uniform(1.5, 3.0),  # Slower, more human-like
                minimum_wait_page_load_time=random.uniform(1.0, 2.0),
                wait_for_network_idle_page_load_time=random.uniform(2.0, 4.0),
                maximum_wait_page_load_time=random.uniform(15.0, 25.0),
                
                # Disable automation hints
                highlight_elements=False,
                
                # Enhanced stealth arguments
                args=[
                    # Language and locale settings
                    "--lang=en-US",
                    "--accept-lang=en-US,en;q=0.9",
                    
                    # Realistic resource usage
                    "--memory-pressure-off",
                    "--max_old_space_size=2048",
                    
                    # Human-like browser features
                    "--enable-features=NetworkService,NetworkServiceInProcess",
                    "--enable-blink-features=HTMLImports",
                    "--force-prefers-reduced-motion",
                    
                    # Disable automation indicators
                    "--disable-background-timer-throttling",
                    "--disable-renderer-backgrounding", 
                    "--disable-backgrounding-occluded-windows",
                    
                    # Enable realistic graphics
                    "--enable-webgl",
                    "--enable-accelerated-2d-canvas",
                    
                    # Additional privacy that looks human
                    "--disable-default-apps",
                    "--disable-sync",
                ]
            )
            
            # Check if we should connect to existing browser
            if cdp_endpoint and cdp_endpoint.startswith('ws://'):
                # Connect to existing browser via WebSocket CDP endpoint
                logger.info(f"🔗 Connecting to existing browser session with enhanced stealth: {cdp_endpoint}")
                
                try:
                    browser_session = BrowserSession(
                        cdp_url=cdp_endpoint,
                        is_local=False,
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
                save_conversation_path=None,
                calculate_cost=True,
            )
            
            logger.info("✅ Ultra-stealth browser-use agent created successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to create ultra-stealth agent: {e}")
            raise
    
    async def run_task(self, task: str, cdp_endpoint: str | None = None, max_steps: int = 10) -> Dict[str, Any]:
        """
        Run the browser automation task using browser-use
        Based on unified-browser-platform architecture
        """
        try:
            # Create agent
            await self.create_agent(task, cdp_endpoint, max_steps)
            
            # Execute the task
            logger.info(f"🎯 Starting task execution: {task}")
            # Check if agent.run() is async or sync
            if asyncio.iscoroutinefunction(self.agent.run):
                result = await self.agent.run()
            else:
                result = self.agent.run()
            
            logger.info("✅ Task completed successfully")
            
            # Return result in the same format as Node.js version
            return {
                "success": True,
                "result": str(result) if result else "Task completed",
                "cdp_endpoint": cdp_endpoint,
                "executed_at": time.time()
            }
            
        except Exception as e:
            logger.error(f"❌ Task execution failed: {e}")
            # Ensure error is JSON serializable
            error_msg = str(e)
            if not error_msg:
                error_msg = "Unknown error occurred"
            
            return {
                "success": False,
                "error": error_msg,
                "error_type": type(e).__name__,
                "cdp_endpoint": cdp_endpoint,
                "executed_at": time.time()
            }

async def main():
    """Main script entry point"""
    if len(sys.argv) < 4:
        print("Usage: python browser_task_fixed.py <session_id> <task> <max_steps> [cdp_endpoint]")
        sys.exit(1)
    
    session_id = sys.argv[1]
    task = sys.argv[2]
    max_steps = int(sys.argv[3])
    cdp_endpoint = sys.argv[4] if len(sys.argv) > 4 else None
    
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