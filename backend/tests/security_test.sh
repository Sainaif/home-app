#!/bin/bash

# Holy Home - Security Penetration Tests
# Tests authentication, authorization, injection, and common vulnerabilities

set -e

API_BASE="${API_BASE:-http://localhost:16162}"
ADMIN_EMAIL="admin@example.pl"
ADMIN_PASSWORD="admin123"

PASSED=0
FAILED=0
VULNERABILITIES=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=============================================="
echo " Holy Home - Security Penetration Test Suite"
echo "=============================================="
echo "Target: $API_BASE"
echo ""

test_pass() {
  echo -e "${GREEN}✓ SECURE${NC} - $1"
  ((PASSED++))
}

test_fail() {
  echo -e "${RED}✗ VULNERABLE${NC} - $1"
  echo "   Details: $2"
  ((FAILED++))
  ((VULNERABILITIES++))
}

test_warning() {
  echo -e "${YELLOW}⚠ WARNING${NC} - $1"
  echo "   Details: $2"
}

# Get admin token
echo "Setting up test environment..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")
ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access":"[^"]*' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo -e "${RED}Failed to authenticate. Exiting.${NC}"
  exit 1
fi
echo -e "${GREEN}Setup complete${NC}"
echo ""

# ==================================================
# 1. Authentication & Session Security Tests
# ==================================================
echo "=== 1. Authentication Security ==="

# Test 1.1: SQL Injection in login
echo -n "1.1 SQL Injection in login... "
SQL_INJECT=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.pl\" OR \"1\"=\"1","password":"anything"}')
if echo "$SQL_INJECT" | grep -q "access"; then
  test_fail "SQL Injection in login" "Login succeeded with SQL injection payload"
else
  test_pass "SQL injection in login prevented"
fi

# Test 1.2: NoSQL Injection in login
echo -n "1.2 NoSQL Injection in login... "
NOSQL_INJECT=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":{"$ne":null},"password":{"$ne":null}}')
if echo "$NOSQL_INJECT" | grep -q "access"; then
  test_fail "NoSQL Injection in login" "Login succeeded with NoSQL injection payload"
else
  test_pass "NoSQL injection in login prevented"
fi

# Test 1.3: Brute force protection
echo -n "1.3 Brute force protection... "
for i in {1..10}; do
  curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@example.pl","password":"wrongpassword"}' > /dev/null
done
BRUTE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.pl","password":"wrongpassword"}')
BRUTE_STATUS=$(echo "$BRUTE_RESPONSE" | tail -n1)
if [ "$BRUTE_STATUS" = "429" ]; then
  test_pass "Rate limiting active (HTTP 429)"
else
  test_warning "Rate limiting not triggered" "Status code: $BRUTE_STATUS (expected 429)"
fi

# Test 1.4: JWT token validation
echo -n "1.4 Invalid JWT rejection... "
INVALID_JWT_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/users" \
  -H "Authorization: Bearer invalidtoken123")
INVALID_JWT_STATUS=$(echo "$INVALID_JWT_RESPONSE" | tail -n1)
if [ "$INVALID_JWT_STATUS" = "401" ]; then
  test_pass "Invalid JWT rejected"
else
  test_fail "Invalid JWT accepted" "Status: $INVALID_JWT_STATUS"
fi

# Test 1.5: Expired token handling (using manipulated token)
echo -n "1.5 Token expiration... "
# Using a token with manipulated exp claim should be rejected
EXPIRED_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiIxMjM0NTYiLCJleHAiOjE1MTYyMzkwMjJ9.invalid"
EXPIRED_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/users" \
  -H "Authorization: Bearer $EXPIRED_TOKEN")
EXPIRED_STATUS=$(echo "$EXPIRED_RESPONSE" | tail -n1)
if [ "$EXPIRED_STATUS" = "401" ]; then
  test_pass "Expired/invalid tokens rejected"
else
  test_warning "Token validation" "May not be checking expiration properly"
fi

# ==================================================
# 2. Authorization & Access Control Tests
# ==================================================
echo ""
echo "=== 2. Authorization & Access Control ==="

# Create a test resident user
TEST_USER_EMAIL="security_test_$(date +%s)@example.com"
CREATE_USER=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_USER_EMAIL\",\"name\":\"Security Test User\",\"password\":\"testpass123\",\"role\":\"RESIDENT\"}")
TEST_USER_ID=$(echo $CREATE_USER | grep -o '"id":"[^"]*' | cut -d'"' -f4)

# Login as resident
RESIDENT_LOGIN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_USER_EMAIL\",\"password\":\"testpass123\"}")
RESIDENT_TOKEN=$(echo $RESIDENT_LOGIN | grep -o '"access":"[^"]*' | cut -d'"' -f4)

# Test 2.1: IDOR - Access other user's data
echo -n "2.1 IDOR protection... "
if [ ! -z "$RESIDENT_TOKEN" ]; then
  # Get list of all users to find another user ID
  USERS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/users")
  OTHER_USER_ID=$(echo $USERS_LIST | grep -o '"id":"[^"]*' | head -2 | tail -1 | cut -d'"' -f4)

  if [ ! -z "$OTHER_USER_ID" ] && [ "$OTHER_USER_ID" != "$TEST_USER_ID" ]; then
    IDOR_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/users/$OTHER_USER_ID" \
      -H "Authorization: Bearer $RESIDENT_TOKEN")
    IDOR_STATUS=$(echo "$IDOR_RESPONSE" | tail -n1)
    if [ "$IDOR_STATUS" = "403" ] || [ "$IDOR_STATUS" = "401" ]; then
      test_pass "IDOR prevented (cannot access other user's data)"
    else
      test_warning "IDOR protection" "User can access other user data (Status: $IDOR_STATUS)"
    fi
  else
    test_warning "IDOR test skipped" "Could not find another user ID"
  fi
fi

# Test 2.2: Privilege escalation
echo -n "2.2 Privilege escalation prevention... "
if [ ! -z "$RESIDENT_TOKEN" ]; then
  PRIV_ESC=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/users" \
    -H "Authorization: Bearer $RESIDENT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"email":"hacker@example.com","name":"Hacker","password":"hack123","role":"ADMIN"}')
  PRIV_STATUS=$(echo "$PRIV_ESC" | tail -n1)
  if [ "$PRIV_STATUS" = "403" ]; then
    test_pass "Privilege escalation prevented"
  else
    test_fail "Privilege escalation possible" "Resident can create users (Status: $PRIV_STATUS)"
  fi
fi

# Test 2.3: Role manipulation
echo -n "2.3 Role manipulation prevention... "
if [ ! -z "$RESIDENT_TOKEN" ] && [ ! -z "$TEST_USER_ID" ]; then
  ROLE_MANIP=$(curl -s -w "\n%{http_code}" -X PATCH "$API_BASE/users/$TEST_USER_ID" \
    -H "Authorization: Bearer $RESIDENT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role":"ADMIN"}')
  ROLE_STATUS=$(echo "$ROLE_MANIP" | tail -n1)
  if [ "$ROLE_STATUS" = "403" ] || [ "$ROLE_STATUS" = "401" ]; then
    test_pass "Role manipulation prevented"
  else
    test_warning "Role manipulation" "User may be able to change their own role"
  fi
fi

# ==================================================
# 3. Input Validation & Injection Tests
# ==================================================
echo ""
echo "=== 3. Input Validation & Injection ==="

# Test 3.1: XSS in user input
echo -n "3.1 XSS prevention... "
XSS_PAYLOAD="<script>alert('XSS')</script>"
XSS_USER=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"xss_test_$(date +%s)@example.com\",\"name\":\"$XSS_PAYLOAD\",\"password\":\"test123\",\"role\":\"RESIDENT\"}")
if echo "$XSS_USER" | grep -q "<script>"; then
  test_warning "XSS prevention" "Script tags not sanitized in response"
else
  test_pass "XSS payload sanitized/escaped"
fi

# Test 3.2: Command injection in notes
echo -n "3.2 Command injection prevention... "
CMD_PAYLOAD="; cat /etc/passwd #"
CMD_INJECT=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"electricity\",\"periodStart\":\"2025-03-01T00:00:00Z\",\"periodEnd\":\"2025-03-31T23:59:59Z\",\"totalAmountPLN\":\"100.00\",\"notes\":\"$CMD_PAYLOAD\"}")
if echo "$CMD_INJECT" | grep -q "root:"; then
  test_fail "Command injection" "Command execution detected in response"
else
  test_pass "Command injection prevented"
fi

# Test 3.3: Path traversal
echo -n "3.3 Path traversal prevention... "
PATH_TRAV=$(curl -s -w "\n%{http_code}" -X GET "$API_BASE/../../../etc/passwd" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
PATH_STATUS=$(echo "$PATH_TRAV" | tail -n1)
if [ "$PATH_STATUS" = "404" ]; then
  test_pass "Path traversal prevented"
else
  test_warning "Path traversal" "Unexpected response: $PATH_STATUS"
fi

# Test 3.4: Large payload (DoS)
echo -n "3.4 Large payload handling... "
LARGE_PAYLOAD=$(python3 -c "print('A' * 1000000)")
LARGE_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"type\":\"electricity\",\"notes\":\"$LARGE_PAYLOAD\"}" \
  --max-time 5)
LARGE_STATUS=$(echo "$LARGE_RESPONSE" | tail -n1)
if [ "$LARGE_STATUS" = "413" ] || [ "$LARGE_STATUS" = "400" ]; then
  test_pass "Large payload rejected"
elif [ "$LARGE_STATUS" = "000" ]; then
  test_warning "Large payload handling" "Request timed out (potential DoS vector)"
else
  test_warning "Large payload handling" "Status: $LARGE_STATUS"
fi

# ==================================================
# 4. Data Security Tests
# ==================================================
echo ""
echo "=== 4. Data Security ==="

# Test 4.1: Password in response
echo -n "4.1 Password exposure... "
USER_DATA=$(curl -s -X GET "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN")
if echo "$USER_DATA" | grep -q "passwordHash\|password_hash\|password"; then
  test_fail "Password exposure" "Password hash visible in API response"
else
  test_pass "Passwords not exposed in API"
fi

# Test 4.2: Weak password acceptance
echo -n "4.2 Weak password prevention... "
WEAK_PASS=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"weak_$(date +%s)@example.com\",\"name\":\"Weak Pass User\",\"password\":\"123\",\"role\":\"RESIDENT\"}")
if echo "$WEAK_PASS" | grep -q "error"; then
  test_pass "Weak passwords rejected"
else
  test_warning "Password policy" "Very weak password (123) was accepted"
fi

# Test 4.3: Email validation
echo -n "4.3 Email validation... "
INVALID_EMAIL=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"not-an-email","name":"Invalid Email","password":"test123","role":"RESIDENT"}')
if echo "$INVALID_EMAIL" | grep -q "error"; then
  test_pass "Invalid emails rejected"
else
  test_warning "Email validation" "Invalid email format accepted"
fi

# ==================================================
# 5. API Security Headers & Configuration
# ==================================================
echo ""
echo "=== 5. Security Headers ==="

# Test 5.1: CORS headers
echo -n "5.1 CORS configuration... "
CORS_RESPONSE=$(curl -s -I -X OPTIONS "$API_BASE/users" \
  -H "Origin: https://evil.com" \
  -H "Access-Control-Request-Method: GET")
if echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Origin: \*"; then
  test_warning "CORS configuration" "Allows all origins (*) - may be intentional for development"
else
  test_pass "CORS properly configured"
fi

# Test 5.2: Security headers
echo -n "5.2 Security headers presence... "
HEADERS=$(curl -s -I "$API_BASE/healthz")
MISSING_HEADERS=""
echo "$HEADERS" | grep -qi "X-Content-Type-Options" || MISSING_HEADERS="$MISSING_HEADERS X-Content-Type-Options"
echo "$HEADERS" | grep -qi "X-Frame-Options" || MISSING_HEADERS="$MISSING_HEADERS X-Frame-Options"
if [ -z "$MISSING_HEADERS" ]; then
  test_pass "Security headers present"
else
  test_warning "Security headers" "Missing:$MISSING_HEADERS"
fi

# Test 5.3: HTTP methods
echo -n "5.3 HTTP method restrictions... "
DELETE_HEALTH=$(curl -s -w "\n%{http_code}" -X DELETE "$API_BASE/healthz")
DELETE_STATUS=$(echo "$DELETE_HEALTH" | tail -n1)
if [ "$DELETE_STATUS" = "405" ]; then
  test_pass "Invalid HTTP methods rejected"
else
  test_warning "HTTP methods" "DELETE on /healthz returned $DELETE_STATUS (expected 405)"
fi

# ==================================================
# Summary
# ==================================================
echo ""
echo "=============================================="
echo " Security Test Results"
echo "=============================================="
echo -e "${GREEN}Secure: $PASSED${NC}"
echo -e "${RED}Vulnerable: $FAILED${NC}"
echo ""

if [ $VULNERABILITIES -gt 0 ]; then
  echo -e "${RED}⚠ $VULNERABILITIES CRITICAL VULNERABILITIES FOUND!${NC}"
  echo "Please review and fix the issues above."
  exit 1
elif [ $FAILED -gt 0 ]; then
  echo -e "${YELLOW}Some security concerns detected.${NC}"
  echo "Review the warnings above."
  exit 0
else
  echo -e "${GREEN}No critical vulnerabilities found!${NC}"
  echo "Application appears secure against common attacks."
  exit 0
fi
