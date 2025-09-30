#!/bin/bash

# Holy Home - Automated API Tests
# Tests all major API endpoints and functionality

set -e

API_BASE="${API_BASE:-http://localhost:16162}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.pl}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

PASSED=0
FAILED=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo " Holy Home - Automated API Test Suite"
echo "========================================="
echo "API: $API_BASE"
echo ""

# Test helper functions
assert_success() {
  local test_name=$1
  local response=$2
  local expected=$3

  if echo "$response" | grep -q "$expected"; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "   Expected: $expected"
    echo "   Got: $response"
    ((FAILED++))
    return 1
  fi
}

assert_status() {
  local test_name=$1
  local status=$2
  local expected=$3

  if [ "$status" = "$expected" ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    ((PASSED++))
    return 0
  else
    echo -e "${RED}✗${NC} $test_name"
    echo "   Expected status: $expected"
    echo "   Got status: $status"
    ((FAILED++))
    return 1
  fi
}

# Test 1: Health Check
echo "=== Health Check Tests ==="
HEALTH_RESPONSE=$(curl -s "$API_BASE/healthz")
assert_success "API health check responds" "$HEALTH_RESPONSE" "ok"

# Test 2: Authentication
echo ""
echo "=== Authentication Tests ==="

# Login with admin
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

ADMIN_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access":"[^"]*' | cut -d'"' -f4)
assert_success "Admin login successful" "$LOGIN_RESPONSE" "access"

# Login with invalid credentials
INVALID_LOGIN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"wrong@example.com","password":"wrongpassword"}')
assert_success "Invalid login rejected" "$INVALID_LOGIN" "error"

# Access protected endpoint without token
UNAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" "$API_BASE/users")
UNAUTH_STATUS=$(echo "$UNAUTH_RESPONSE" | tail -n1)
assert_status "Unauthorized access rejected" "$UNAUTH_STATUS" "401"

# Test 3: User Management
echo ""
echo "=== User Management Tests ==="

# List users
USERS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/users")
assert_success "List users" "$USERS_LIST" "email"

# Create a test user
TEST_USER_EMAIL="test_$(date +%s)@example.com"
CREATE_USER=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_USER_EMAIL\",\"name\":\"Test User\",\"password\":\"testpass123\",\"role\":\"RESIDENT\"}")
TEST_USER_ID=$(echo $CREATE_USER | grep -o '"id":"[^"]*' | cut -d'"' -f4)
assert_success "Create user" "$CREATE_USER" "id"

# Get user by ID
if [ ! -z "$TEST_USER_ID" ]; then
  GET_USER=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/users/$TEST_USER_ID")
  assert_success "Get user by ID" "$GET_USER" "$TEST_USER_EMAIL"
fi

# Test 4: Group Management
echo ""
echo "=== Group Management Tests ==="

# Create group
TEST_GROUP_NAME="Test Group $(date +%s)"
CREATE_GROUP=$(curl -s -X POST "$API_BASE/groups" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$TEST_GROUP_NAME\",\"weight\":1.5}")
TEST_GROUP_ID=$(echo $CREATE_GROUP | grep -o '"id":"[^"]*' | cut -d'"' -f4)
assert_success "Create group" "$CREATE_GROUP" "id"

# List groups
GROUPS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/groups")
assert_success "List groups" "$GROUPS_LIST" "$TEST_GROUP_NAME"

# Test 5: Bill Management
echo ""
echo "=== Bill Management Tests ==="

# Create bill
CREATE_BILL=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"electricity",
    "periodStart":"2025-03-01T00:00:00Z",
    "periodEnd":"2025-03-31T23:59:59Z",
    "totalAmountPLN":"400.00",
    "totalUnits":"500.0",
    "notes":"Test electricity bill",
    "status":"draft"
  }')
TEST_BILL_ID=$(echo $CREATE_BILL | grep -o '"id":"[^"]*' | cut -d'"' -f4)
assert_success "Create bill" "$CREATE_BILL" "draft"

# List bills
BILLS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/bills")
assert_success "List bills" "$BILLS_LIST" "electricity"

# Get bill by ID
if [ ! -z "$TEST_BILL_ID" ]; then
  GET_BILL=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/bills/$TEST_BILL_ID")
  assert_success "Get bill by ID" "$GET_BILL" "Test electricity bill"
fi

# Test 6: Consumption Management
echo ""
echo "=== Consumption Management Tests ==="

if [ ! -z "$TEST_BILL_ID" ] && [ ! -z "$TEST_USER_ID" ]; then
  # Create consumption
  CREATE_CONSUMPTION=$(curl -s -X POST "$API_BASE/consumptions" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"billId\":\"$TEST_BILL_ID\",
      \"userId\":\"$TEST_USER_ID\",
      \"units\":\"100.5\",
      \"meterValue\":\"5000.0\",
      \"recordedAt\":\"2025-03-31T20:00:00Z\"
    }")
  assert_success "Create consumption reading" "$CREATE_CONSUMPTION" "units"

  # List consumptions
  CONSUMPTIONS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/consumptions")
  assert_success "List consumptions" "$CONSUMPTIONS_LIST" "billId"
fi

# Test 7: Loan Management
echo ""
echo "=== Loan Management Tests ==="

# Get existing users for loan test
USERS_RESPONSE=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/users")
USER1_ID=$(echo $USERS_RESPONSE | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
USER2_ID=$(echo $USERS_RESPONSE | grep -o '"id":"[^"]*' | head -2 | tail -1 | cut -d'"' -f4)

if [ ! -z "$USER1_ID" ] && [ ! -z "$USER2_ID" ] && [ "$USER1_ID" != "$USER2_ID" ]; then
  # Create loan
  CREATE_LOAN=$(curl -s -X POST "$API_BASE/loans" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"lenderId\":\"$USER1_ID\",
      \"borrowerId\":\"$USER2_ID\",
      \"amountPLN\":\"150.00\",
      \"note\":\"Test loan\",
      \"dueDate\":\"2025-04-15T00:00:00Z\"
    }")
  TEST_LOAN_ID=$(echo $CREATE_LOAN | grep -o '"id":"[^"]*' | cut -d'"' -f4)
  assert_success "Create loan" "$CREATE_LOAN" "open"

  # List loans
  LOANS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/loans")
  assert_success "List loans" "$LOANS_LIST" "lenderId"

  # Get balances
  BALANCES=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/loans/balances")
  assert_success "Get loan balances" "$BALANCES" "balances"
fi

# Test 8: Chore Management
echo ""
echo "=== Chore Management Tests ==="

# Create chore
CREATE_CHORE=$(curl -s -X POST "$API_BASE/chores" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Test Chore",
    "description":"Automated test chore",
    "frequency":"weekly",
    "difficulty":3,
    "priority":3,
    "assignmentMode":"manual",
    "notificationsEnabled":false,
    "isActive":true
  }')
TEST_CHORE_ID=$(echo $CREATE_CHORE | grep -o '"id":"[^"]*' | cut -d'"' -f4)
assert_success "Create chore" "$CREATE_CHORE" "Test Chore"

# List chores
CHORES_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/chores")
assert_success "List chores" "$CHORES_LIST" "name"

# Test 9: Chore Assignments
echo ""
echo "=== Chore Assignment Tests ==="

if [ ! -z "$TEST_CHORE_ID" ] && [ ! -z "$TEST_USER_ID" ]; then
  # Create chore assignment
  CREATE_ASSIGNMENT=$(curl -s -X POST "$API_BASE/chore-assignments" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"choreId\":\"$TEST_CHORE_ID\",
      \"assigneeUserId\":\"$TEST_USER_ID\",
      \"dueDate\":\"2025-03-10T18:00:00Z\",
      \"status\":\"pending\"
    }")
  assert_success "Create chore assignment" "$CREATE_ASSIGNMENT" "pending"

  # List chore assignments
  ASSIGNMENTS_LIST=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_BASE/chore-assignments")
  assert_success "List chore assignments" "$ASSIGNMENTS_LIST" "choreId"
fi

# Test 10: Token Refresh
echo ""
echo "=== Token Refresh Tests ==="

REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"refresh":"[^"]*' | cut -d'"' -f4)
if [ ! -z "$REFRESH_TOKEN" ]; then
  REFRESH_RESPONSE=$(curl -s -X POST "$API_BASE/auth/refresh" \
    -H "Content-Type: application/json" \
    -d "{\"refreshToken\":\"$REFRESH_TOKEN\"}")
  assert_success "Token refresh" "$REFRESH_RESPONSE" "access"
fi

# Test 11: RBAC - Resident permissions
echo ""
echo "=== RBAC Tests ==="

# Login as resident user (if exists)
RESIDENT_LOGIN=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"anna@example.com","password":"password123"}')
RESIDENT_TOKEN=$(echo $RESIDENT_LOGIN | grep -o '"access":"[^"]*' | cut -d'"' -f4)

if [ ! -z "$RESIDENT_TOKEN" ]; then
  # Try to create user as resident (should fail)
  RESIDENT_CREATE_USER=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/users" \
    -H "Authorization: Bearer $RESIDENT_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"email":"forbidden@example.com","name":"Forbidden","password":"test123","role":"RESIDENT"}')
  RESIDENT_STATUS=$(echo "$RESIDENT_CREATE_USER" | tail -n1)
  assert_status "Resident cannot create users" "$RESIDENT_STATUS" "403"
fi

# Test 12: Data Validation
echo ""
echo "=== Data Validation Tests ==="

# Try to create bill with invalid data
INVALID_BILL=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"invalid_type",
    "periodStart":"invalid_date",
    "totalAmountPLN":"not_a_number"
  }')
assert_success "Invalid bill data rejected" "$INVALID_BILL" "error"

# Try to create user with duplicate email
if [ ! -z "$TEST_USER_EMAIL" ]; then
  DUPLICATE_USER=$(curl -s -X POST "$API_BASE/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_USER_EMAIL\",\"name\":\"Duplicate\",\"password\":\"test123\",\"role\":\"RESIDENT\"}")
  assert_success "Duplicate email rejected" "$DUPLICATE_USER" "error"
fi

# Summary
echo ""
echo "========================================="
echo " Test Results"
echo "========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo "Total: $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}Some tests failed!${NC}"
  exit 1
fi
