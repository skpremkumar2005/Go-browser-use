# Clean Browser Automation Platform

A clean, simple browser automation platform using Go backend, Python worker with browser-use, and real-time streaming.

## Features

- ✅ Clean Go server with session management
- ✅ Python worker using browser-use library
- ✅ Real-time browser streaming via CDP
- ✅ Simple web frontend
- ✅ Dynamic port allocation
- ✅ No conflicts or legacy code

## Quick Start

### 1. Install Dependencies

**Go:**
```bash
go mod tidy
```

**Python:**
```bash
pip install -r requirements.txt
```

### 2. Configure Environment

Create a `.env` file for browser-use LLM configuration:
```bash
# For Azure OpenAI (Recommended)
AZURE_OPENAI_API_KEY=your_azure_openai_api_key_here
AZURE_OPENAI_ENDPOINT=https://your-resource-name.openai.azure.com/
AZURE_OPENAI_API_VERSION=2024-02-15-preview
AZURE_OPENAI_DEPLOYMENT_NAME=your-deployment-name

# Or for OpenAI
# OPENAI_API_KEY=your_openai_api_key_here
```

**Azure OpenAI Setup:**
1. Go to Azure Portal → Create Azure OpenAI resource
2. Deploy a model (e.g., gpt-4, gpt-3.5-turbo)
3. Copy your API key, endpoint, and deployment name
4. Update the `.env` file with your credentials

### 3. Start Services

**Terminal 1 - Python Worker:**
```bash
python python_worker.py
```

**Terminal 2 - Go Server:**
```bash
go run main.go
```

### 4. Open Frontend

Visit: http://localhost:8080

## Usage

1. Enter a task in the frontend (e.g., "Go to YouTube and search for WebRTC")
2. Click "Start Task"
3. Watch the real-time browser stream
4. Monitor session status

## Architecture

```
Frontend (HTML/JS) → Go Server (Port 8080) → Python Worker (Port 8081) → Chrome (Dynamic CDP Port)
```

## API Endpoints

- `POST /api/tasks` - Start new task
- `GET /api/sessions/{id}` - Get session details
- `GET /api/sessions` - List all sessions
- `GET /api/stream/{id}` - Stream browser content

## Clean Design Principles

- ✅ No legacy code
- ✅ No conflicts
- ✅ Simple, readable code
- ✅ Proper error handling
- ✅ Clean separation of concerns
- ✅ Modern libraries only

## Troubleshooting

1. **Chrome not launching**: Ensure chromium-browser is installed
2. **LLM errors**: Check your .env configuration
3. **Port conflicts**: The system automatically finds free ports
4. **Streaming issues**: Check browser CDP port in session details

## Development

This is a clean, minimal implementation focused on:
- Working browser automation
- Real-time streaming
- Simple session management
- No unnecessary complexity# gobrowseruse
# Go-browser-use
# Go-browser-use
# Go-browser-use
# Go-browser-use
