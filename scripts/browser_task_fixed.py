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
        """Load and validate environment variables with support for different LLM providers"""
        try:
            # Check for new LLM configuration from environment variables (set by Go backend)
            llm_provider = os.getenv("LLM_PROVIDER")
            llm_api_key = os.getenv("LLM_API_KEY")
            llm_endpoint = os.getenv("LLM_ENDPOINT")
            llm_deployment = os.getenv("LLM_DEPLOYMENT")
            llm_model = os.getenv("LLM_MODEL")
            
            if all([llm_provider, llm_api_key, llm_endpoint, llm_deployment, llm_model]):
                logger.info(f"✅ Using LLM configuration from Go backend: {llm_provider} - {llm_model}")
                self._configure_llm_from_env(llm_provider, llm_api_key, llm_endpoint, llm_deployment, llm_model)
                return
            
            # Fallback to legacy Azure OpenAI configuration from environment
            api_key = os.getenv("AZURE_OPENAI_API_KEY")
            api_base = os.getenv("AZURE_OPENAI_API_BASE") or os.getenv("AZURE_OPENAI_ENDPOINT")
            api_version = os.getenv("AZURE_OPENAI_API_VERSION", "2024-08-01-preview")  # Updated default version
            deployment_name = os.getenv("AZURE_OPENAI_DEPLOYMENT_NAME")
            
            if not all([api_key, api_base, deployment_name]):
                logger.error("❌ Missing required LLM configuration")
                logger.error("Either provide LLM_* variables from Go backend or Azure OpenAI variables")
                logger.error("Required: AZURE_OPENAI_API_KEY, AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT_NAME")
                sys.exit(1)
            
            # Import ChatAzureOpenAI for proper Azure OpenAI configuration
            from browser_use.llm import ChatAzureOpenAI
            
            # Create ChatAzureOpenAI for Azure OpenAI (fallback configuration)
            self.llm = ChatAzureOpenAI(
                model=deployment_name,
                api_key=api_key,
                azure_endpoint=api_base,
                azure_deployment=deployment_name,
                api_version=api_version,
                temperature=0.7,
                max_retries=3,  # Add retry for rate limiting
                timeout=60.0,   # Increase timeout
            )
            
            logger.info("✅ Azure OpenAI LLM configured successfully (fallback)")
            
        except Exception as e:
            logger.error(f"❌ Failed to configure LLM: {e}")
            sys.exit(1)
    
    def _configure_llm_from_env(self, provider: str, api_key: str, endpoint: str, deployment: str, model: str):
        """Configure LLM based on provider from Go backend environment variables"""
        try:
            if provider.lower() == "azure":
                from browser_use.llm import ChatAzureOpenAI
                
                # Extract API version from endpoint if present, otherwise use default
                api_version = "2024-08-01-preview"  # Updated to newer version that supports json_schema
                if "api-version=" in endpoint:
                    import urllib.parse
                    parsed_url = urllib.parse.urlparse(endpoint)
                    query_params = urllib.parse.parse_qs(parsed_url.query)
                    if "api-version" in query_params:
                        api_version = query_params["api-version"][0]
                        # Ensure we use a compatible version
                        if api_version < "2024-08-01-preview":
                            logger.warning(f"⚠️ API version {api_version} may not support all features, using 2024-08-01-preview")
                            api_version = "2024-08-01-preview"
                
                self.llm = ChatAzureOpenAI(
                    model=model,
                    api_key=api_key,
                    azure_endpoint=endpoint,
                    azure_deployment=deployment,
                    api_version=api_version,
                    temperature=0.7,
                    max_retries=3,  # Add retry for rate limiting
                    timeout=60.0,   # Increase timeout
                )
                logger.info(f"✅ Azure OpenAI LLM configured: {model} at {endpoint}")
                
            elif provider.lower() == "openai":
                from browser_use.llm import ChatOpenAI
                
                self.llm = ChatOpenAI(
                    model=model,
                    api_key=api_key,
                    base_url=endpoint if endpoint != "https://api.openai.com/v1" else None,
                    temperature=0.7
                )
                logger.info(f"✅ OpenAI LLM configured: {model}")
                
            elif provider.lower() == "anthropic":
                from browser_use.llm import ChatAnthropic
                
                self.llm = ChatAnthropic(
                    model=model,
                    api_key=api_key,
                    base_url=endpoint if endpoint != "https://api.anthropic.com" else None,
                    temperature=0.7
                )
                logger.info(f"✅ Anthropic LLM configured: {model}")
                
            else:
                logger.error(f"❌ Unsupported LLM provider: {provider}")
                logger.error("Supported providers: azure, openai, anthropic")
                sys.exit(1)
                
        except ImportError as e:
            logger.error(f"❌ Failed to import LLM class for {provider}: {e}")
            sys.exit(1)
        except Exception as e:
            logger.error(f"❌ Failed to configure {provider} LLM: {e}")
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
                    # For existing browsers, create a browser session that connects to the CDP endpoint
                    browser_session = BrowserSession(
                        cdp_url=cdp_endpoint,
                        is_local=False,
                        browser_profile=browser_profile
                    )
                    
                    logger.info(f"✅ Connected to existing browser session for streaming integration")
                    
                except Exception as e:
                    logger.error(f"❌ Failed to connect to existing browser session: {e}")
                    logger.info(f"🆕 Falling back to creating new browser instance")
                    # Fallback to creating new browser
                    browser_session = BrowserSession(browser_profile=browser_profile)
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
            # Always await the agent's run method since it's async
            result = await self.agent.run()
            
            logger.info("✅ Task completed successfully")
            
            # Get CDP endpoint from the browser session that was created
            actual_cdp_endpoint = cdp_endpoint
            if self.agent and hasattr(self.agent, 'browser_session') and self.agent.browser_session:
                # Try to extract CDP URL from browser session - browser_use creates its own browser
                browser_session = self.agent.browser_session

                # Check various possible attributes for CDP URL
                if hasattr(browser_session, 'cdp_url') and browser_session.cdp_url:
                    actual_cdp_endpoint = browser_session.cdp_url
                elif hasattr(browser_session, '_cdp_url') and browser_session._cdp_url:
                    actual_cdp_endpoint = browser_session._cdp_url
                elif hasattr(browser_session, 'connection_url') and browser_session.connection_url:
                    actual_cdp_endpoint = browser_session.connection_url
                elif hasattr(browser_session, '_connection_url') and browser_session._connection_url:
                    actual_cdp_endpoint = browser_session._connection_url
                elif hasattr(browser_session, 'browser_context') and browser_session.browser_context:
                    # Try to get CDP endpoint from browser context
                    ctx = browser_session.browser_context
                    if hasattr(ctx, 'cdp_session') and hasattr(ctx.cdp_session, '_ws_url'):
                        actual_cdp_endpoint = ctx.cdp_session._ws_url

                # Also try to get the debugger URL from browser process
                if not actual_cdp_endpoint and hasattr(browser_session, '_browser_process'):
                    process = browser_session._browser_process
                    if hasattr(process, 'debugger_url'):
                        actual_cdp_endpoint = process.debugger_url

                # Try to construct CDP endpoint from localhost port if we can find it
                if not actual_cdp_endpoint:
                    # Look for CDP port in logs or try common port range
                    import re
                    # Check if we can extract port from any internal URLs
                    for attr_name in dir(browser_session):
                        if not attr_name.startswith('_'):
                            continue
                        attr_val = getattr(browser_session, attr_name, None)
                        if isinstance(attr_val, str) and 'localhost:' in attr_val and '/devtools/' in attr_val:
                            actual_cdp_endpoint = attr_val
                            break

                # NEW: Try to get CDP endpoint from browser profile or internal state
                if not actual_cdp_endpoint:
                    # Check browser profile for CDP information
                    if hasattr(browser_session, 'browser_profile') and browser_session.browser_profile:
                        profile = browser_session.browser_profile
                        if hasattr(profile, 'cdp_url') and profile.cdp_url:
                            actual_cdp_endpoint = profile.cdp_url
                        elif hasattr(profile, '_cdp_url') and profile._cdp_url:
                            actual_cdp_endpoint = profile._cdp_url

                # NEW: Try to access the underlying browser connection
                if not actual_cdp_endpoint:
                    try:
                        # Try to get the browser context and check for connection details
                        if hasattr(browser_session, 'browser_context') and browser_session.browser_context:
                            ctx = browser_session.browser_context
                            # Check if there's a connection or session with URL info
                            if hasattr(ctx, 'connection') and ctx.connection:
                                conn = ctx.connection
                                if hasattr(conn, 'url') and conn.url:
                                    actual_cdp_endpoint = conn.url
                                elif hasattr(conn, '_url') and conn._url:
                                    actual_cdp_endpoint = conn._url
                    except Exception as e:
                        logger.debug(f"Could not access browser connection: {e}")

                # NEW: Try to find the port from browser process arguments
                if not actual_cdp_endpoint:
                    try:
                        # Check if we can find the remote debugging port from the browser process
                        import psutil
                        import os

                        # Get current process and look for chrome processes
                        current_pid = os.getpid()
                        parent = psutil.Process(current_pid)

                        # Look for chrome processes in the process tree
                        for proc in parent.children(recursive=True):
                            try:
                                if 'chrome' in proc.name().lower() or 'chromium' in proc.name().lower():
                                    # Get command line arguments
                                    cmdline = proc.cmdline()
                                    for arg in cmdline:
                                        if '--remote-debugging-port=' in arg:
                                            port = arg.split('=')[1]
                                            actual_cdp_endpoint = f"ws://localhost:{port}/devtools/browser"
                                            logger.info(f"🔍 Found Chrome debugging port: {port}")
                                            break
                                    if actual_cdp_endpoint:
                                        break
                            except (psutil.NoSuchProcess, psutil.AccessDenied):
                                continue
                    except ImportError:
                        logger.debug("psutil not available for process inspection")
                    except Exception as e:
                        logger.debug(f"Could not inspect browser processes: {e}")

                # NEW: Try to extract CDP endpoint from browser-use logs or network requests
                if not actual_cdp_endpoint:
                    try:
                        # Try to get the CDP endpoint by making a request to the browser's JSON endpoint
                        # This is what browser-use does internally
                        import httpx
                        import json

                        # Try common ports that browser-use might use
                        common_ports = [59035, 58926, 9222, 9223, 9224, 9225]
                        for port in common_ports:
                            try:
                                response = httpx.get(f"http://localhost:{port}/json/version", timeout=1.0)
                                if response.status_code == 200:
                                    version_data = response.json()
                                    if 'webSocketDebuggerUrl' in version_data:
                                        actual_cdp_endpoint = version_data['webSocketDebuggerUrl']
                                        logger.info(f"🔗 Found CDP endpoint via HTTP: {actual_cdp_endpoint}")
                                        break
                            except (httpx.RequestError, json.JSONDecodeError):
                                continue
                    except ImportError:
                        logger.debug("httpx not available for HTTP requests")
                    except Exception as e:
                        logger.debug(f"Could not get CDP endpoint via HTTP: {e}")

                # Log the CDP endpoint for Go to capture
                if actual_cdp_endpoint:
                    logger.info(f"🔗 Browser CDP endpoint found: {actual_cdp_endpoint}")
                else:
                    logger.warning("⚠️ Could not extract CDP endpoint from browser session")
                    # Try to get it from environment or use a fallback
                    # This is a last resort - browser-use should provide the endpoint
                    logger.info("🔍 Browser session attributes available:")
                    for attr in dir(browser_session):
                        if not attr.startswith('__'):
                            logger.info(f"  - {attr}: {type(getattr(browser_session, attr, None))}")

            # Return result in the same format as Node.js version
            return {
                "success": True,
                "result": str(result) if result else "Task completed",
                "cdp_endpoint": actual_cdp_endpoint,
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