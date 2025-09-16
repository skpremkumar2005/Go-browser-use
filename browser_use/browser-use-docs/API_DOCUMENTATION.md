# Browser-Use API Documentation

## Overview

Browser-Use is an AI agent that autonomously interacts with the web. It takes a user-defined task, navigates web pages using Chromium via CDP, processes HTML, and repeatedly queries a language model to decide the next action until the task is completed.

## Core Components

### 1. Agent Class

The main entry point for creating and running browser automation tasks.

```python
from browser_use import Agent, ChatOpenAI

# Create an agent
agent = Agent(
    task="Your task description",
    llm=ChatOpenAI(model="gpt-4o"),
    controller=controller,
    calculate_cost=True  # Enable token cost tracking
)

# Run the agent
history = await agent.run()
```

#### Agent Parameters

- `task` (str): The task description for the AI agent
- `llm`: Language model instance (OpenAI, Anthropic, etc.)
- `controller` (Controller): Browser controller instance
- `calculate_cost` (bool): Enable token cost tracking
- `max_steps` (int): Maximum number of steps for the agent
- `system_prompt` (str): Custom system prompt
- `output_format` (dict): Structured output format

### 2. Controller

Manages browser actions and provides a registry of available actions.

```python
from browser_use.core.controller import Controller

controller = Controller()

# Register custom actions
@controller.registry.action("Custom action description")
async def custom_action(param1: str, param2: int, page: Page):
    # Implementation
    return ActionResult(extracted_content=result, include_in_memory=True)
```

#### Built-in Actions

- `click_element`: Click on a specific element
- `type_text`: Type text into an input field
- `navigate_to_url`: Navigate to a specific URL
- `scroll_page`: Scroll the page up/down
- `extract_text`: Extract text from elements
- `wait_for_element`: Wait for an element to appear
- `take_screenshot`: Capture a screenshot

### 3. Browser Session

Manages browser instances and provides CDP connectivity.

```python
from browser_use.browser.session import BrowserSession

# Create a browser session
session = BrowserSession(
    headless=False,
    viewport={"width": 1920, "height": 1080},
    user_agent="Custom User Agent"
)

# Connect to existing browser
session = BrowserSession.from_cdp_endpoint("ws://localhost:9222")
```

#### Session Parameters

- `headless` (bool): Run browser in headless mode
- `viewport` (dict): Browser viewport dimensions
- `user_agent` (str): Custom user agent string
- `proxy` (str): Proxy server configuration
- `downloads_path` (str): Downloads directory path

## Language Model Integrations

### OpenAI

```python
from browser_use.llm.openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o",
    api_key="your-api-key",
    temperature=0.1
)
```

### Anthropic (Claude)

```python
from browser_use.llm.anthropic import ChatAnthropic

llm = ChatAnthropic(
    model="claude-3-sonnet-20240229",
    api_key="your-api-key"
)
```

### Azure OpenAI

```python
from browser_use.llm.openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o",
    api_key="your-azure-api-key",
    base_url="https://your-resource.openai.azure.com/openai/deployments/your-deployment",
    api_version="2024-02-15-preview"
)
```

### Google (Gemini)

```python
from browser_use.llm.google import ChatGoogle

llm = ChatGoogle(
    model="gemini-1.5-pro",
    api_key="your-api-key"
)
```

### AWS Bedrock

```python
from browser_use.llm.aws import ChatBedrock

llm = ChatBedrock(
    model="anthropic.claude-3-sonnet-20240229-v1:0",
    region="us-east-1"
)
```

## Custom Actions

### Creating Custom Actions

```python
from browser_use.core.controller import Controller, ActionResult
from playwright.async_api import Page

controller = Controller()

@controller.registry.action("Search for products on e-commerce site")
async def search_products(query: str, page: Page):
    # Navigate to search page
    await page.goto("https://example.com/search")

    # Fill search input
    await page.fill("#search-input", query)
    await page.click("#search-button")

    # Extract results
    results = await page.query_selector_all(".product-item")
    product_data = []

    for result in results:
        title = await result.query_selector(".product-title")
        price = await result.query_selector(".product-price")

        product_data.append({
            "title": await title.text_content() if title else "",
            "price": await price.text_content() if price else ""
        })

    return ActionResult(
        extracted_content=product_data,
        include_in_memory=True
    )
```

### Action Parameters

- `page` (Page): Playwright page object (always required)
- Custom parameters: Any additional parameters your action needs

### ActionResult

- `extracted_content`: Data extracted from the action
- `include_in_memory`: Whether to include in agent's memory
- `screenshot`: Optional screenshot data
- `error`: Error message if action failed

## Token Cost Tracking

### Enabling Cost Tracking

```python
from browser_use import Agent

agent = Agent(
    task="Your task",
    llm=llm,
    calculate_cost=True  # Enable tracking
)
```

### Accessing Cost Data

```python
# Get cost summary
cost_summary = agent.token_cost_service.get_summary()

# Get detailed usage
usage_history = agent.token_cost_service.usage_history

# Get cost for specific model
model_cost = agent.token_cost_service.get_model_cost("gpt-4o")
```

## Configuration

### Environment Variables

```bash
# OpenAI
OPENAI_API_KEY=your-api-key

# Anthropic
ANTHROPIC_API_KEY=your-api-key

# Azure OpenAI
AZURE_OPENAI_API_KEY=your-api-key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=your-deployment-name

# Google
GOOGLE_API_KEY=your-api-key

# AWS
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-east-1
```

### Browser Configuration

```python
from browser_use.browser.profile import BrowserProfile

profile = BrowserProfile(
    headless=False,
    viewport={"width": 1920, "height": 1080},
    user_agent="Custom User Agent",
    proxy="http://proxy:8080",
    downloads_path="./downloads"
)
```

## Error Handling

### Common Exceptions

```python
from browser_use.exceptions import (
    BrowserUseException,
    NavigationError,
    ElementNotFoundError,
    ActionFailedError
)

try:
    await agent.run()
except ElementNotFoundError as e:
    print(f"Element not found: {e}")
except NavigationError as e:
    print(f"Navigation failed: {e}")
except ActionFailedError as e:
    print(f"Action failed: {e}")
```

## Advanced Features

### Custom System Prompts

```python
agent = Agent(
    task="Your task",
    llm=llm,
    system_prompt="""
    You are a specialized web automation agent.
    Focus on efficiency and accuracy.
    Always verify actions before executing them.
    """
)
```

### Structured Output

```python
agent = Agent(
    task="Extract product information",
    llm=llm,
    output_format={
        "type": "object",
        "properties": {
            "products": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"},
                        "price": {"type": "string"},
                        "url": {"type": "string"}
                    }
                }
            }
        }
    }
)
```

### Multi-Tab Support

```python
# Create multiple tabs
tab1 = await session.new_tab()
tab2 = await session.new_tab()

# Switch between tabs
await session.switch_to_tab(tab1)
await session.switch_to_tab(tab2)
```

### File Downloads

```python
# Configure download path
session = BrowserSession(downloads_path="./downloads")

# Monitor downloads
downloads = await session.get_downloads()
for download in downloads:
    print(f"Downloaded: {download.filename}")
```

## Integration Examples

### Web Scraping

```python
from browser_use import Agent, ChatOpenAI

async def scrape_website():
    agent = Agent(
        task="Scrape all product information from the homepage",
        llm=ChatOpenAI(model="gpt-4o"),
        calculate_cost=True
    )

    result = await agent.run()
    return result.extracted_content
```

### Form Filling

```python
async def fill_contact_form():
    agent = Agent(
        task="Fill out the contact form with the provided information",
        llm=ChatOpenAI(model="gpt-4o")
    )

    result = await agent.run()
    return result
```

### E-commerce Automation

```python
async def purchase_product(product_url, quantity=1):
    agent = Agent(
        task=f"Purchase {quantity} items from {product_url}",
        llm=ChatOpenAI(model="gpt-4o"),
        max_steps=50
    )

    result = await agent.run()
    return result
```

## Best Practices

1. **Error Handling**: Always wrap agent execution in try-catch blocks
2. **Resource Management**: Close browser sessions when done
3. **Rate Limiting**: Implement delays between actions when needed
4. **Validation**: Verify extracted data before using it
5. **Logging**: Enable logging for debugging and monitoring
6. **Cost Tracking**: Monitor token usage to control costs
7. **Security**: Use environment variables for API keys
8. **Testing**: Test actions in isolation before running full tasks

## Troubleshooting

### Common Issues

1. **Browser Connection Failed**

   - Check if Chrome/Chromium is installed
   - Verify CDP endpoint is accessible
   - Check firewall settings

2. **Element Not Found**

   - Wait for page to load completely
   - Use more specific selectors
   - Check if element is in iframe

3. **Navigation Timeout**

   - Increase timeout settings
   - Check network connectivity
   - Verify URL is accessible

4. **Token Cost Not Tracking**
   - Ensure `calculate_cost=True`
   - Check API key permissions
   - Verify model pricing is available

### Debug Mode

```python
import logging

# Enable debug logging
logging.basicConfig(level=logging.DEBUG)

# Run agent with debug info
agent = Agent(task="Your task", llm=llm)
result = await agent.run()
```
