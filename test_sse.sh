#!/bin/bash

# Test SSE connection with proper authentication

API_URL="http://localhost:16162"
EMAIL="${ADMIN_EMAIL:-admin@example.pl}"
PASSWORD="${ADMIN_PASSWORD:-admin123}"

echo "=== Testing SSE Connection ==="
echo

# Login and get access token
echo "1. Logging in as $EMAIL..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access":"[^"]*' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
  echo "❌ Login failed!"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo "✅ Login successful, got access token"
echo "Token: ${ACCESS_TOKEN:0:20}..."
echo

# Test SSE connection
echo "2. Connecting to SSE stream..."
echo "   (Will show events for 10 seconds, then exit)"
echo

timeout 10 curl -N -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$API_URL/events/stream" 2>&1 || true

echo
echo
echo "=== Test Complete ==="
