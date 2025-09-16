# Server Deployment Guide

## 🚨 HARDCODED VALUES FIXED & DEPLOYMENT READY

All hardcoded values have been identified and made configurable through environment variables. The application is now ready for server deployment.

## ✅ FIXED HARDCODED VALUES:

### 1. **IP Addresses & Hosts**
- ❌ Before: `127.0.0.1` hardcoded in CDP connections
- ✅ Fixed: `CDP_HOST` environment variable (default: 127.0.0.1)
- ✅ Fixed: `SERVER_HOST` configurable (set to `0.0.0.0` for server deployment)

### 2. **CORS Origins**  
- ❌ Before: Only `localhost` and `127.0.0.1` allowed
- ✅ Fixed: `ALLOWED_ORIGINS` environment variable supports any domains

### 3. **Chrome/Browser Path**
- ❌ Before: Hardcoded `google-chrome` path
- ✅ Fixed: `CHROME_PATH` environment variable + intelligent path detection
- ✅ Added: Automatic detection of Chrome/Chromium on server systems

### 4. **CDP Port Range**
- ❌ Before: Hardcoded port array `[9222, 9223, ...]`
- ✅ Fixed: `CDP_PORTS` environment variable (comma-separated)

### 5. **URL Generation**
- ✅ Already dynamic: Uses request host and protocol detection
- ✅ Supports HTTPS via `X-Forwarded-Proto` header (proxy-friendly)

## 🚀 SERVER DEPLOYMENT STEPS:

### 1. **Copy Server Configuration**
```bash
cp .env.server .env
```

### 2. **Configure Environment Variables**
Edit `.env` and configure these REQUIRED variables:

**Server Configuration:**
```bash
SERVER_HOST=0.0.0.0  # Listen on all interfaces
ALLOWED_ORIGINS=https://yourdomain.com,http://yourdomain.com:3000
```

**AI Configuration (Choose one):**
```bash
# Option 1: Azure OpenAI
AZURE_OPENAI_API_KEY=your_key_here
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_DEPLOYMENT_NAME=gpt-4

# Option 2: Google AI
LLM_PROVIDER=google  
GOOGLE_API_KEY=your_google_key_here

# Option 3: Anthropic
LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your_anthropic_key_here
```

**Security:**
```bash
API_KEYS=your_secure_api_key_here
```

### 3. **Install Dependencies**

**Ubuntu/Debian:**
```bash
# Install Chrome/Chromium
sudo apt update
sudo apt install -y google-chrome-stable

# Or Chromium as alternative
sudo apt install -y chromium-browser

# Install Python and dependencies
sudo apt install -y python3 python3-pip
pip3 install browser-use
```

**CentOS/RHEL:**
```bash
# Install Chrome
wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | sudo rpm --import -
sudo yum install -y google-chrome-stable

# Install Python and dependencies  
sudo yum install -y python3 python3-pip
pip3 install browser-use
```

**For Headless Servers (Docker/VPS):**
```bash
# Install Xvfb for virtual display
sudo apt install -y xvfb

# Start virtual display (add to startup scripts)
export DISPLAY=:99
Xvfb :99 -screen 0 1920x1080x24 &
```

### 4. **Build and Run**
```bash
cd cmd/script-server
go build -o browser-automation-server main.go
./browser-automation-server
```

Or run directly:
```bash
cd cmd/script-server  
go run main.go
```

### 5. **Systemd Service (Production)**
Create `/etc/systemd/system/browser-automation.service`:
```ini
[Unit]
Description=Browser Automation Server
After=network.target

[Service]
Type=simple
User=automation
WorkingDirectory=/opt/browser-automation/cmd/script-server
ExecStart=/opt/browser-automation/cmd/script-server/browser-automation-server
Restart=always
RestartSec=5
Environment=DISPLAY=:99

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable browser-automation
sudo systemctl start browser-automation
sudo systemctl status browser-automation
```

## 🔧 ENVIRONMENT VARIABLES REFERENCE:

### **Core Server Configuration**
- `PORT` - Server port (default: 3000)
- `SERVER_HOST` - Bind address (use `0.0.0.0` for server)
- `ALLOWED_ORIGINS` - CORS allowed origins (comma-separated)

### **Chrome/Browser Configuration**
- `CHROME_PATH` - Chrome executable path (auto-detected if empty)
- `CDP_HOST` - CDP connection host (default: 127.0.0.1)
- `CDP_PORTS` - CDP ports to try (default: 9222,9223,9224,...)
- `BROWSER_HEADLESS` - Headless mode (set `true` for servers)

### **AI Configuration**
- `LLM_PROVIDER` - AI provider: "azure_openai", "google", "anthropic"
- `AZURE_OPENAI_API_KEY` - Azure OpenAI API key
- `AZURE_OPENAI_ENDPOINT` - Azure OpenAI endpoint URL
- `GOOGLE_API_KEY` - Google AI API key
- `ANTHROPIC_API_KEY` - Anthropic API key

### **Security**
- `API_KEYS` - API keys for authentication (comma-separated)

### **Display (Headless Servers)**
- `DISPLAY` - X11 display (default: :99)
- `XVFB_DISPLAY` - Xvfb display (default: :99)

## 🧪 TESTING SERVER DEPLOYMENT:

### 1. **Health Check**
```bash
curl http://your-server:3000/api/status
```

### 2. **Simple Automation Test**
```bash
curl -X POST http://your-server:3000/api/task \
  -H "Content-Type: application/json" \
  -d '{"task": "Go to Google.com", "max_steps": 5}'
```

### 3. **Enhanced Stealth Test**  
```bash
curl -X POST http://your-server:3000/api/task \
  -H "Content-Type: application/json" \
  -d '{"task": "Search Google for Python tutorial", "max_steps": 10}'
```

## 🐳 DOCKER DEPLOYMENT:

### Dockerfile Example:
```dockerfile
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    golang-go \
    google-chrome-stable \
    python3 \
    python3-pip \
    xvfb \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install browser-use

WORKDIR /app
COPY . .

WORKDIR /app/cmd/script-server
RUN go build -o server main.go

EXPOSE 3000

ENV DISPLAY=:99
ENV BROWSER_HEADLESS=true
ENV SERVER_HOST=0.0.0.0

CMD ["sh", "-c", "Xvfb :99 -screen 0 1920x1080x24 & ./server"]
```

## 🔒 SECURITY RECOMMENDATIONS:

1. **Use HTTPS in production**
2. **Configure firewall to restrict port 3000 access**
3. **Use strong API keys**
4. **Regularly update Chrome and dependencies**
5. **Monitor logs for suspicious activity**
6. **Use reverse proxy (nginx) for SSL termination**

## 📋 READY FOR DEPLOYMENT ✅

Your application is now fully configurable and ready for server deployment with no hardcoded values!