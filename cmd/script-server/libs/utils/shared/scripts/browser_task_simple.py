#!/usr/bin/env python3
"""
Simple browser task script for testing without LLM dependencies
"""

import asyncio
import json
import logging
import sys
import time
import websockets

# Setup logging
logging.basicConfig(level=logging.INFO, format='%(levelname)s:%(name)s:%(message)s')
logger = logging.getLogger(__name__)

class SimpleBrowserTask:
    def __init__(self, session_id: str, task: str, max_steps: int, cdp_endpoint: str):
        self.session_id = session_id
        self.task = task
        self.max_steps = max_steps
        self.cdp_endpoint = cdp_endpoint
        self.websocket = None
    
    async def connect_to_browser(self):
        """Connect to the existing browser via CDP"""
        try:
            logger.info(f"🔗 Connecting to browser: {self.cdp_endpoint}")
            self.websocket = await websockets.connect(self.cdp_endpoint)
            logger.info("✅ Connected to browser")
            
            # Enable Page domain
            await self.send_command("Page.enable")
            logger.info("✅ Enabled Page domain")
            
            return True
        except Exception as e:
            logger.error(f"❌ Failed to connect to browser: {e}")
            return False
    
    async def send_command(self, method: str, params: dict = None):
        """Send a command to CDP"""
        request = {
            "id": 1,
            "method": method,
            "params": params or {}
        }
        
        await self.websocket.send(json.dumps(request))
        response = await self.websocket.recv()
        return json.loads(response)
    
    async def navigate_to_google(self):
        """Navigate to Google"""
        try:
            logger.info("🌐 Navigating to Google...")
            response = await self.send_command("Page.navigate", {
                "url": "https://www.google.com"
            })
            
            if "error" not in response:
                logger.info("✅ Successfully navigated to Google")
                return True
            else:
                logger.error(f"❌ Navigation failed: {response.get('error')}")
                return False
        except Exception as e:
            logger.error(f"❌ Navigation error: {e}")
            return False
    
    async def wait_for_page_load(self):
        """Wait for page to load"""
        try:
            logger.info("⏳ Waiting for page to load...")
            await asyncio.sleep(3)  # Wait 3 seconds for page to load
            
            # Check if page is loaded
            response = await self.send_command("Runtime.evaluate", {
                "expression": "document.readyState"
            })
            
            if "result" in response and "result" in response["result"]:
                ready_state = response["result"]["result"]["value"]
                logger.info(f"📄 Page ready state: {ready_state}")
                return ready_state == "complete"
            
            return True
        except Exception as e:
            logger.error(f"❌ Error waiting for page load: {e}")
            return False
    
    async def execute_task(self):
        """Execute the browser task"""
        try:
            logger.info(f"🚀 Starting task: {self.task}")
            
            # Connect to browser
            if not await self.connect_to_browser():
                return False
            
            # Navigate to Google
            if not await self.navigate_to_google():
                return False
            
            # Wait for page to load
            if not await self.wait_for_page_load():
                return False
            
            logger.info("✅ Task completed successfully")
            return True
            
        except Exception as e:
            logger.error(f"❌ Task execution failed: {e}")
            return False
        finally:
            if self.websocket:
                await self.websocket.close()
                logger.info("🔌 Disconnected from browser")

async def main():
    """Main function"""
    if len(sys.argv) < 5:
        logger.error("Usage: python browser_task_simple.py <session_id> <task> <max_steps> <cdp_endpoint>")
        sys.exit(1)
    
    session_id = sys.argv[1]
    task = sys.argv[2]
    max_steps = int(sys.argv[3])
    cdp_endpoint = sys.argv[4]
    
    logger.info(f"🚀 Starting simple browser task script")
    logger.info(f"📋 Session ID: {session_id}")
    logger.info(f"🎯 Task: {task}")
    logger.info(f"📊 Max Steps: {max_steps}")
    logger.info(f"🔌 CDP Endpoint: {cdp_endpoint}")
    
    # Create and execute task
    browser_task = SimpleBrowserTask(session_id, task, max_steps, cdp_endpoint)
    success = await browser_task.execute_task()
    
    # Return result
    result = {
        "success": success,
        "session_id": session_id,
        "task": task,
        "message": "Task completed successfully" if success else "Task failed",
        "timestamp": time.time()
    }
    
    print(json.dumps(result))
    
    if not success:
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())


