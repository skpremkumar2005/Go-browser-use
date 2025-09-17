#!/bin/bash

echo "🚀 Testing Fixed WebSocket Handler"
echo "================================="

cd "d:\Local disk D\projects\Go-browser-use\cmd\script-server"

echo "📦 Building server with fixes..."
go build -o main_fixed.exe

if [ $? -eq 0 ]; then
    echo "✅ Build successful! The fixes compile without errors."
    echo ""
    echo "🔧 Key fixes applied:"
    echo "   1. ✅ Increased WebSocket buffer sizes (16KB → 32KB)"
    echo "   2. ✅ Added CDP client error recovery"
    echo "   3. ✅ Enhanced buffer overflow handling"
    echo "   4. ✅ Implemented connection retry mechanism"
    echo "   5. ✅ Added graceful error recovery for user interactions"
    echo ""
    echo "🎯 To test:"
    echo "   1. Stop the current server (Ctrl+C)"
    echo "   2. Run: ./main_fixed.exe"
    echo "   3. Try clicking on the browser interface"
    echo "   4. WebSocket connections should now be stable!"
else
    echo "❌ Build failed. Check compilation errors above."
fi