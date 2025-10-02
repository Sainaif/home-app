#!/bin/bash

# Test SSE connection for 20 seconds to verify heartbeats

API_URL="http://localhost:16162"
EMAIL="admin@example.pl"
PASSWORD="admin123"

echo "=== Testing SSE Connection (20 seconds) ==="
echo

# Login and get access token
echo "1. Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access":"[^"]*' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
  echo "❌ Login failed!"
  exit 1
fi

echo "✅ Login successful"
echo

# Test SSE connection for 20 seconds
echo "2. Connecting to SSE (will receive heartbeats every 15s)..."
echo "   Press Ctrl+C to stop"
echo "---"

timeout 20 curl -N -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$API_URL/events/stream" 2>&1 | while IFS= read -r line; do
    # Only show data lines, skip curl progress
    if [[ "$line" =~ ^(data:|event:|id:|:) ]]; then
      timestamp=$(date +"%H:%M:%S")
      echo "[$timestamp] $line"
    fi
  done

echo "---"
echo
echo "✅ Test complete"
