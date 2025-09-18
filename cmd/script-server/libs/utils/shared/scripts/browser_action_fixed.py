#!/usr/bin/env python3
"""
Fixed Browser Action Script with proper CDP integration
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
    """Direct Chrome DevTools Protocol client for browser interactions"""
    
    def __init__(self, cdp_endpoint: str):
        self.cdp_endpoint = cdp_endpoint
        self.websocket = None
        self.request_id = 0
        self.viewport_width = 1920
        self.viewport_height = 1080
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
            await self.send_command("Input.enable")
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
            
            # Wait for response
            response = await self.websocket.recv()
            return json.loads(response)
    
    async def click(self, x: float, y: float):
        """Click at coordinates (x, y) as percentages of viewport"""
        try:
            # Convert percentages to pixels
            pixel_x = int(x * self.viewport_width)
            pixel_y = int(y * self.viewport_height)
            
            logger.info(f"🖱️ Clicking at ({pixel_x}, {pixel_y})")
            
            # Send mouse events
            await self.send_command("Input.dispatchMouseEvent", {
                "type": "mousePressed",
                "x": pixel_x,
                "y": pixel_y,
                "button": "left",
                "clickCount": 1
            })
            
            await self.send_command("Input.dispatchMouseEvent", {
                "type": "mouseReleased",
                "x": pixel_x,
                "y": pixel_y,
                "button": "left",
                "clickCount": 1
            })
            
            logger.info("✅ Click executed successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to click: {e}")
            raise
    
    async def type_text(self, text: str):
        """Type text into the current focused element"""
        try:
            logger.info(f"⌨️ Typing text: {text}")
            
            # Send key events for each character
            for char in text:
                await self.send_command("Input.dispatchKeyEvent", {
                    "type": "keyDown",
                    "text": char
                })
                await self.send_command("Input.dispatchKeyEvent", {
                    "type": "keyUp",
                    "text": char
                })
            
            logger.info("✅ Text typed successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to type text: {e}")
            raise
    
    async def press_key(self, key: str):
        """Press a key"""
        try:
            logger.info(f"⌨️ Pressing key: {key}")
            
            # Map common keys to CDP key codes
            key_map = {
                "Enter": "Enter",
                "Tab": "Tab",
                "Escape": "Escape",
                "Backspace": "Backspace",
                "Delete": "Delete",
                "ArrowUp": "ArrowUp",
                "ArrowDown": "ArrowDown",
                "ArrowLeft": "ArrowLeft",
                "ArrowRight": "ArrowRight",
                "Home": "Home",
                "End": "End",
                "PageUp": "PageUp",
                "PageDown": "PageDown"
            }
            
            cdp_key = key_map.get(key, key)
            
            await self.send_command("Input.dispatchKeyEvent", {
                "type": "keyDown",
                "key": cdp_key
            })
            await self.send_command("Input.dispatchKeyEvent", {
                "type": "keyUp",
                "key": cdp_key
            })
            
            logger.info("✅ Key pressed successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to press key: {e}")
            raise
    
    async def scroll(self, direction: str, pixels: int):
        """Scroll the page"""
        try:
            delta_x = 0
            delta_y = pixels if direction == "down" else -pixels
            
            logger.info(f"📜 Scrolling {direction} by {pixels} pixels")
            
            await self.send_command("Input.dispatchMouseEvent", {
                "type": "mouseWheel",
                "x": self.viewport_width // 2,
                "y": self.viewport_height // 2,
                "deltaX": delta_x,
                "deltaY": delta_y
            })
            
            logger.info("✅ Scroll executed successfully")
            
        except Exception as e:
            logger.error(f"❌ Failed to scroll: {e}")
            raise
    
    async def disconnect(self):
        """Disconnect from CDP"""
        if self.websocket:
            await self.websocket.close()
            logger.info("🔌 Disconnected from CDP")

async def execute_action(action: str, data: Dict[str, Any], cdp_endpoint: str) -> bool:
    """Execute a browser action"""
    try:
        # Connect to CDP
        cdp_client = CDPClient(cdp_endpoint)
        connected = await cdp_client.connect()
        if not connected:
            raise Exception("Failed to connect to CDP")
        
        try:
            # Execute the action
            if action == "click":
                x = data.get("x", 0.5)
                y = data.get("y", 0.5)
                await cdp_client.click(x, y)
            elif action == "type":
                text = data.get("text", "")
                if not text:
                    raise Exception("No text provided for type action")
                await cdp_client.type_text(text)
            elif action == "key":
                key = data.get("key", "")
                if not key:
                    raise Exception("No key provided for key action")
                await cdp_client.press_key(key)
            elif action == "scroll":
                direction = data.get("direction", "down")
                pixels = data.get("pixels", 300)
                await cdp_client.scroll(direction, pixels)
            else:
                raise Exception(f"Unknown action: {action}")
            
            logger.info(f"✅ Successfully executed {action} action")
            return True
            
        finally:
            await cdp_client.disconnect()
            
    except Exception as e:
        logger.error(f"❌ Error executing {action} action: {e}")
        return False

async def main():
    """Main script entry point"""
    if len(sys.argv) < 4:
        print("Usage: python browser_action_fixed.py <action> <data_json> <cdp_endpoint>")
        sys.exit(1)
    
    action = sys.argv[1]
    data_json = sys.argv[2]
    cdp_endpoint = sys.argv[3]
    
    try:
        data = json.loads(data_json)
    except json.JSONDecodeError as e:
        print(f"Error parsing data JSON: {e}")
        sys.exit(1)
    
    logger.info(f"🚀 Starting browser action script")
    logger.info(f"🎯 Action: {action}")
    logger.info(f"📊 Data: {data}")
    logger.info(f"🔌 CDP Endpoint: {cdp_endpoint}")
    
    # Execute action
    success = await execute_action(action, data, cdp_endpoint)
    
    # Output result to stdout for Go to read
    result = {
        "success": success,
        "action": action,
        "data": data,
        "executed_at": time.time()
    }
    
    print(json.dumps(result))
    
    if success:
        logger.info(f"✅ Action executed successfully")
        sys.exit(0)
    else:
        logger.error(f"❌ Action execution failed")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())