# Browser-Use Setup Guide

## Overview

This guide provides step-by-step instructions for setting up the Browser-Use library on both Windows and Linux systems.

## Prerequisites

### System Requirements

- **Python**: 3.11 or higher
- **Chrome/Chromium**: Latest stable version
- **Memory**: Minimum 4GB RAM (8GB recommended)
- **Storage**: 2GB free space

### Required Software

1. **Python 3.11+**
2. **Git**
3. **Chrome/Chromium Browser**
4. **uv** (Python package manager)

## Windows Setup

### Step 1: Install Python

1. **Download Python 3.11+**

   ```bash
   # Visit https://www.python.org/downloads/
   # Download Python 3.11 or higher for Windows
   ```

2. **Install Python**

   - Run the installer
   - Check "Add Python to PATH"
   - Check "Install for all users" (recommended)

3. **Verify Installation**
   ```cmd
   python --version
   # Should show Python 3.11.x or higher
   ```

### Step 2: Install Git

1. **Download Git**

   ```bash
   # Visit https://git-scm.com/download/win
   # Download Git for Windows
   ```

2. **Install Git**

   - Run the installer
   - Use default settings
   - Add Git to PATH

3. **Verify Installation**
   ```cmd
   git --version
   # Should show Git version
   ```

### Step 3: Install Chrome/Chromium

1. **Download Chrome**

   ```bash
   # Visit https://www.google.com/chrome/
   # Download and install Chrome
   ```

2. **Verify Installation**
   ```cmd
   # Chrome should be available in PATH
   chrome --version
   ```

### Step 4: Install uv

1. **Install uv using pip**

   ```cmd
   pip install uv
   ```

2. **Verify Installation**
   ```cmd
   uv --version
   ```

### Step 5: Clone and Setup Browser-Use

1. **Clone Repository**

   ```cmd
   git clone https://github.com/browser-use/browser-use.git
   cd browser-use
   ```

2. **Create Virtual Environment**

   ```cmd
   uv venv --python 3.11
   ```

3. **Activate Virtual Environment**

   ```cmd
   .venv\Scripts\activate
   ```

4. **Install Dependencies**

   ```cmd
   uv sync
   ```

5. **Install Browser-Use**
   ```cmd
   uv pip install -e .
   ```

### Step 6: Configure Environment Variables

1. **Create Environment File**

   ```cmd
   copy .env.example .env
   ```

2. **Edit Environment Variables**
   ```cmd
   # Open .env file and add your API keys
   OPENAI_API_KEY=your-openai-api-key
   AZURE_OPENAI_API_KEY=your-azure-api-key
   AZURE_OPENAI_ENDPOINT=your-azure-endpoint
   AZURE_OPENAI_DEPLOYMENT=your-deployment-name
   ANTHROPIC_API_KEY=your-anthropic-api-key
   ```

### Step 7: Test Installation

1. **Run Basic Test**

   ```cmd
   python -c "from browser_use import Agent; print('Browser-Use installed successfully!')"
   ```

2. **Test Browser Connection**
   ```cmd
   python examples/getting_started/01_basic_search.py
   ```

## Linux Setup

### Step 1: Update System

```bash
# Ubuntu/Debian
sudo apt update && sudo apt upgrade -y

# CentOS/RHEL
sudo yum update -y

# Fedora
sudo dnf update -y
```

### Step 2: Install Python

#### Ubuntu/Debian

```bash
# Install Python 3.11
sudo apt install python3.11 python3.11-venv python3.11-dev python3-pip

# Install build dependencies
sudo apt install build-essential libssl-dev libffi-dev python3-dev
```

#### CentOS/RHEL

```bash
# Install Python 3.11
sudo yum install python3.11 python3.11-devel python3-pip

# Install build dependencies
sudo yum groupinstall "Development Tools"
sudo yum install openssl-devel libffi-devel
```

#### Fedora

```bash
# Install Python 3.11
sudo dnf install python3.11 python3.11-devel python3-pip

# Install build dependencies
sudo dnf groupinstall "Development Tools"
sudo dnf install openssl-devel libffi-devel
```

### Step 3: Install Git

```bash
# Ubuntu/Debian
sudo apt install git

# CentOS/RHEL
sudo yum install git

# Fedora
sudo dnf install git
```

### Step 4: Install Chrome/Chromium

#### Ubuntu/Debian

```bash
# Install Chromium
sudo apt install chromium-browser

# Or install Chrome
wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" | sudo tee /etc/apt/sources.list.d/google-chrome.list
sudo apt update
sudo apt install google-chrome-stable
```

#### CentOS/RHEL

```bash
# Install Chromium
sudo yum install chromium

# Or install Chrome
sudo yum install wget
wget https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm
sudo yum install google-chrome-stable_current_x86_64.rpm
```

#### Fedora

```bash
# Install Chromium
sudo dnf install chromium

# Or install Chrome
sudo dnf install wget
wget https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm
sudo dnf install google-chrome-stable_current_x86_64.rpm
```

### Step 5: Install uv

```bash
# Install uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.cargo/bin:$PATH"

# Reload shell
source ~/.bashrc
```

### Step 6: Clone and Setup Browser-Use

1. **Clone Repository**

   ```bash
   git clone https://github.com/browser-use/browser-use.git
   cd browser-use
   ```

2. **Create Virtual Environment**

   ```bash
   uv venv --python 3.11
   ```

3. **Activate Virtual Environment**

   ```bash
   source .venv/bin/activate
   ```

4. **Install Dependencies**

   ```bash
   uv sync
   ```

5. **Install Browser-Use**
   ```bash
   uv pip install -e .
   ```

### Step 7: Configure Environment Variables

1. **Create Environment File**

   ```bash
   cp .env.example .env
   ```

2. **Edit Environment Variables**
   ```bash
   nano .env
   # Add your API keys
   ```

### Step 8: Test Installation

1. **Run Basic Test**

   ```bash
   python -c "from browser_use import Agent; print('Browser-Use installed successfully!')"
   ```

2. **Test Browser Connection**
   ```bash
   python examples/getting_started/01_basic_search.py
   ```

## Docker Setup

### Option 1: Using Docker Compose

1. **Clone Repository**

   ```bash
   git clone https://github.com/browser-use/browser-use.git
   cd browser-use
   ```

2. **Build and Run**
   ```bash
   docker-compose up --build
   ```

### Option 2: Using Dockerfile

1. **Build Image**

   ```bash
   docker build -t browser-use .
   ```

2. **Run Container**
   ```bash
   docker run -it --rm \
     -e OPENAI_API_KEY=your-api-key \
     -v $(pwd):/app \
     browser-use
   ```

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# OpenAI Configuration
OPENAI_API_KEY=your-openai-api-key

# Azure OpenAI Configuration
AZURE_OPENAI_API_KEY=your-azure-api-key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT=your-deployment-name

# Anthropic Configuration
ANTHROPIC_API_KEY=your-anthropic-api-key

# Google Configuration
GOOGLE_API_KEY=your-google-api-key

# AWS Configuration
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key
AWS_REGION=us-east-1

# Browser Configuration
BROWSER_HEADLESS=false
BROWSER_VIEWPORT_WIDTH=1920
BROWSER_VIEWPORT_HEIGHT=1080

# Logging Configuration
LOG_LEVEL=INFO
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

## Quick Start Examples

### Basic Search Example

```python
from browser_use import Agent, ChatOpenAI

async def basic_search():
    # Create LLM instance
    llm = ChatOpenAI(
        model="gpt-4o",
        api_key="your-api-key"
    )

    # Create agent
    agent = Agent(
        task="Search for 'Python programming' on Google and return the first 3 results",
        llm=llm,
        calculate_cost=True
    )

    # Run agent
    result = await agent.run()
    print(result.extracted_content)

# Run the example
import asyncio
asyncio.run(basic_search())
```

### Form Filling Example

```python
async def fill_form():
    llm = ChatOpenAI(model="gpt-4o", api_key="your-api-key")

    agent = Agent(
        task="Fill out the contact form at https://example.com/contact with the following information: Name: John Doe, Email: john@example.com, Message: Hello World",
        llm=llm
    )

    result = await agent.run()
    print("Form filled successfully!")

asyncio.run(fill_form())
```

## Troubleshooting

### Common Issues

#### 1. Python Version Issues

```bash
# Check Python version
python --version

# If not 3.11+, install correct version
# Windows: Download from python.org
# Linux: Use package manager or pyenv
```

#### 2. Chrome/Chromium Not Found

```bash
# Windows: Ensure Chrome is in PATH
# Linux: Install chromium-browser or google-chrome-stable

# Verify installation
chrome --version
# or
chromium-browser --version
```

#### 3. Permission Issues (Linux)

```bash
# Fix permission issues
sudo chown -R $USER:$USER ~/.cache/pip
sudo chown -R $USER:$USER ~/.local

# If using Docker, run with proper permissions
docker run -it --rm \
  -v $(pwd):/app \
  -u $(id -u):$(id -g) \
  browser-use
```

#### 4. API Key Issues

```bash
# Verify API keys are set
echo $OPENAI_API_KEY
echo $AZURE_OPENAI_API_KEY

# Test API connection
python -c "
import openai
openai.api_key = 'your-api-key'
print('API key is valid')
"
```

#### 5. Memory Issues

```bash
# Increase swap space (Linux)
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Add to /etc/fstab for persistence
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### Debug Mode

```python
import logging

# Enable debug logging
logging.basicConfig(level=logging.DEBUG)

# Run with debug info
from browser_use import Agent
agent = Agent(task="Your task", llm=llm)
result = await agent.run()
```

### Getting Help

1. **Check Documentation**: Visit the official documentation
2. **GitHub Issues**: Search existing issues or create new ones
3. **Discord Community**: Join the Discord server for help
4. **Stack Overflow**: Search for browser-use related questions

## Performance Optimization

### System Optimization

1. **Increase Memory**: Ensure at least 8GB RAM
2. **SSD Storage**: Use SSD for better I/O performance
3. **CPU Cores**: Multi-core CPU recommended
4. **Network**: Stable internet connection for API calls

### Browser Optimization

```python
# Optimize browser settings
profile = BrowserProfile(
    headless=True,  # Faster execution
    viewport={"width": 1280, "height": 720},  # Smaller viewport
    args=[
        "--disable-gpu",
        "--disable-dev-shm-usage",
        "--no-sandbox",
        "--disable-setuid-sandbox"
    ]
)
```

### API Optimization

```python
# Use caching for repeated requests
from browser_use.llm.openai import ChatOpenAI

llm = ChatOpenAI(
    model="gpt-4o",
    api_key="your-api-key",
    temperature=0.1,  # Lower temperature for consistent results
    max_tokens=1000   # Limit token usage
)
```

## Security Considerations

1. **API Keys**: Never commit API keys to version control
2. **Environment Variables**: Use .env files for local development
3. **Network Security**: Use HTTPS for all API communications
4. **Browser Security**: Run in isolated environment when possible
5. **Input Validation**: Validate all user inputs before processing

## Next Steps

After successful installation:

1. **Read the Documentation**: Explore the API documentation
2. **Try Examples**: Run through the example scripts
3. **Build Your Own**: Create custom automation scripts
4. **Join Community**: Connect with other users
5. **Contribute**: Help improve the project
