#!/usr/bin/env python3
import os
import sys

print("Python Environment Test")
print("=" * 40)
print(f"AZURE_OPENAI_API_KEY: {'SET' if os.getenv('AZURE_OPENAI_API_KEY') else 'NOT SET'}")
print(f"AZURE_OPENAI_ENDPOINT: {os.getenv('AZURE_OPENAI_ENDPOINT', 'NOT SET')}")
print(f"AZURE_OPENAI_DEPLOYMENT_NAME: {os.getenv('AZURE_OPENAI_DEPLOYMENT_NAME', 'NOT SET')}")
print(f"Python path: {sys.executable}")
print(f"Args: {sys.argv}")
print("=" * 40)