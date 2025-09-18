#!/usr/bin/env python3
"""
Fixed Browser Screenshot Script with proper CDP integration
Connects to existing browser session like the Node.js version
"""

import asyncio
import json
import logging
import sys
import time
import websockets
import aiohttp
from typing import Optional, Dict, Any

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class CDPClient:
    """Direct Chrome DevTools Protocol client for screenshots"""
    
    def __init__(self, cdp_endpoint: str):
        self.cdp_endpoint = cdp_endpoint
        self.websocket = None
        self.request_id = 0
        self._lock = asyncio.Lock()
    
    async def connect(self):
        """Connect to Chrome CDP WebSocket"""
        try:
            # If cdp_endpoint is a WebSocket URL, use it directly
            if self.cdp_endpoint.startswith('ws://'):
                websocket_url = self.cdp_endpoint
            else:
                # Extract port and construct WebSocket URL
                port = self.cdp_endpoint
                async with aiohttp.ClientSession() as session:
                    async with session.get(f'http://127.0.0.1:{port}/json') as resp:
                        if resp.status == 200:
                            tabs = await resp.json()
                            if tabs and len(tabs) > 0:
                                websocket_url = tabs[0].get('webSocketDebuggerUrl')
                            else:
                                raise Exception("No tabs found")
                        else:
                            raise Exception(f"Failed to get tabs: {resp.status}")
            
            if not websocket_url:
                raise Exception("No WebSocket URL found")
            
            self.websocket = await websockets.connect(
                websocket_url,
                ping_interval=20,
                ping_timeout=10,
                close_timeout=10
            )
            logger.info(f"🔗 Connected to CDP: {websocket_url}")
            
            # Enable available domains
            await self.send_command("Page.enable")
            await self.send_command("Runtime.enable")
            await self.send_command("DOM.enable")
            logger.info("✅ Enabled CDP domains")
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to connect to CDP: {e}")
            return False
    
    async def send_command(self, method: str, params: Dict[str, Any] = None) -> Dict[str, Any]:
        """Send a command to CDP"""
        async with self._lock:
            self.request_id += 1
            request = {
                "id": self.request_id,
                "method": method,
                "params": params or {}
            }
            
            await self.websocket.send(json.dumps(request))
            
            # Wait for response - filter out event notifications
            while True:
                response = await self.websocket.recv()
                response_data = json.loads(response)
                
                # Check if this is a command response (has 'id' field matching our request)
                if "id" in response_data and response_data["id"] == self.request_id:
                    return response_data
                # If it's an event notification (has 'method' field), ignore it and continue
                elif "method" in response_data:
                    logger.debug(f"🔍 Ignoring event: {response_data['method']}")
                    continue
                else:
                    # Unknown response format, return it anyway
                    logger.warning(f"⚠️ Unknown response format: {response_data}")
                    return response_data
    
    async def capture_screenshot(self) -> str:
        """Capture a screenshot using CDP"""
        try:
            # Capture screenshot
            response = await self.send_command("Page.captureScreenshot", {
                "format": "jpeg",
                "quality": 95
            })
            
            logger.info(f"🔍 Screenshot response: {json.dumps(response, indent=2)}")
            
            if "result" in response and "data" in response["result"]:
                screenshot_data = response["result"]["data"]
                logger.info(f"📸 Captured screenshot ({len(screenshot_data)} bytes)")
                return screenshot_data
            else:
                logger.error(f"❌ Invalid response structure: {response}")
                raise Exception("No screenshot data in response")
                
        except Exception as e:
            logger.error(f"❌ Failed to capture screenshot: {e}")
            raise
    
    async def disconnect(self):
        """Disconnect from CDP"""
        if self.websocket:
            await self.websocket.close()
            logger.info("🔌 Disconnected from CDP")

async def capture_screenshot_from_cdp(cdp_endpoint: str) -> str:
    """Capture screenshot from CDP endpoint"""
    try:
        # Connect to CDP
        cdp_client = CDPClient(cdp_endpoint)
        connected = await cdp_client.connect()
        if not connected:
            raise Exception("Failed to connect to CDP")
        
        try:
            # Capture screenshot
            screenshot_data = await cdp_client.capture_screenshot()
            return screenshot_data
            
        finally:
            await cdp_client.disconnect()
            
    except Exception as e:
        logger.error(f"❌ Error capturing screenshot: {e}")
        raise

async def main():
    """Main script entry point"""
    if len(sys.argv) < 2:
        print("Usage: python browser_screenshot_fixed.py <cdp_endpoint>")
        sys.exit(1)
    
    cdp_endpoint = sys.argv[1]
    
    logger.info(f"🚀 Starting browser screenshot script")
    logger.info(f"🔌 CDP Endpoint: {cdp_endpoint}")
    
    try:
        # Capture screenshot from CDP endpoint
        screenshot_data = await capture_screenshot_from_cdp(cdp_endpoint)
        
        # Output result to stdout for Go to read
        result = {
            "success": True,
            "screenshot": screenshot_data,
            "captured_at": time.time()
        }
        
        print(json.dumps(result))
        logger.info("✅ Screenshot captured successfully")
        sys.exit(0)
        
    except Exception as e:
        result = {
            "success": False,
            "error": str(e),
            "captured_at": time.time()
        }
        
        print(json.dumps(result))
        logger.error(f"❌ Screenshot capture failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())