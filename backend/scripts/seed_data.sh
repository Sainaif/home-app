#!/bin/bash

# Holy Home - Comprehensive Seed Data Script
# Creates realistic test data for bills, users, groups, loans, and chores

set -e

API_BASE="${API_BASE:-http://localhost:16162}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

echo "=== Holy Home Seed Data Script ==="
echo "API: $API_BASE"
echo ""

# Login as admin
echo "1. Logging in as admin..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to get access token"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo "✓ Logged in successfully"
echo ""

# Create groups
echo "2. Creating household groups..."
GROUP1=$(curl -s -X POST "$API_BASE/groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Anna i Piotr","weight":2.0}' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

GROUP2=$(curl -s -X POST "$API_BASE/groups" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Maria","weight":1.0}' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

echo "✓ Created 2 groups"
echo ""

# Create users
echo "3. Creating users..."
USER1=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"anna@example.com\",\"name\":\"Anna Kowalska\",\"password\":\"password123\",\"role\":\"RESIDENT\",\"groupId\":\"$GROUP1\"}" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

USER2=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"piotr@example.com\",\"name\":\"Piotr Kowalski\",\"password\":\"password123\",\"role\":\"RESIDENT\",\"groupId\":\"$GROUP1\"}" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

USER3=$(curl -s -X POST "$API_BASE/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"maria@example.com\",\"name\":\"Maria Nowak\",\"password\":\"password123\",\"role\":\"RESIDENT\",\"groupId\":\"$GROUP2\"}" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

echo "✓ Created 3 resident users"
echo ""

# Create electricity bills for the last 3 months
echo "4. Creating electricity bills..."
BILL1=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"electricity",
    "periodStart":"2024-12-01T00:00:00Z",
    "periodEnd":"2024-12-31T23:59:59Z",
    "totalAmountPLN":"345.67",
    "totalUnits":"456.8",
    "notes":"December electricity bill",
    "status":"closed"
  }' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

BILL2=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"electricity",
    "periodStart":"2025-01-01T00:00:00Z",
    "periodEnd":"2025-01-31T23:59:59Z",
    "totalAmountPLN":"412.34",
    "totalUnits":"523.2",
    "notes":"January electricity bill",
    "status":"posted"
  }' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

BILL3=$(curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"electricity",
    "periodStart":"2025-02-01T00:00:00Z",
    "periodEnd":"2025-02-28T23:59:59Z",
    "totalAmountPLN":"389.12",
    "totalUnits":"487.5",
    "notes":"February electricity bill",
    "status":"draft"
  }' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

echo "✓ Created 3 electricity bills"

# Create gas bills
echo "5. Creating gas bills..."
curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"gas",
    "periodStart":"2025-01-01T00:00:00Z",
    "periodEnd":"2025-01-31T23:59:59Z",
    "totalAmountPLN":"156.78",
    "totalUnits":"78.4",
    "notes":"January gas bill",
    "status":"closed"
  }' > /dev/null

curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"gas",
    "periodStart":"2025-02-01T00:00:00Z",
    "periodEnd":"2025-02-28T23:59:59Z",
    "totalAmountPLN":"189.45",
    "totalUnits":"94.7",
    "notes":"February gas bill",
    "status":"posted"
  }' > /dev/null

echo "✓ Created 2 gas bills"

# Create internet bills
echo "6. Creating internet bills..."
curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"internet",
    "periodStart":"2025-01-01T00:00:00Z",
    "periodEnd":"2025-01-31T23:59:59Z",
    "totalAmountPLN":"89.00",
    "notes":"January internet - 500 Mbps",
    "status":"closed"
  }' > /dev/null

curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"internet",
    "periodStart":"2025-02-01T00:00:00Z",
    "periodEnd":"2025-02-28T23:59:59Z",
    "totalAmountPLN":"89.00",
    "notes":"February internet - 500 Mbps",
    "status":"posted"
  }' > /dev/null

echo "✓ Created 2 internet bills"

# Create other bills
echo "7. Creating other bills..."
curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"inne",
    "customType":"Water",
    "periodStart":"2025-01-01T00:00:00Z",
    "periodEnd":"2025-01-31T23:59:59Z",
    "totalAmountPLN":"67.50",
    "totalUnits":"15.3",
    "notes":"January water bill",
    "status":"closed"
  }' > /dev/null

curl -s -X POST "$API_BASE/bills" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"inne",
    "customType":"Trash",
    "periodStart":"2025-02-01T00:00:00Z",
    "periodEnd":"2025-02-28T23:59:59Z",
    "totalAmountPLN":"45.00",
    "notes":"February trash collection",
    "status":"posted"
  }' > /dev/null

echo "✓ Created 2 other bills"

# Add consumption readings for electricity bill 3
echo "8. Adding consumption readings..."
if [ ! -z "$USER1" ] && [ ! -z "$BILL3" ]; then
  curl -s -X POST "$API_BASE/consumptions" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"billId\":\"$BILL3\",
      \"userId\":\"$USER1\",
      \"units\":\"125.5\",
      \"meterValue\":\"12345.6\",
      \"recordedAt\":\"2025-02-28T20:00:00Z\"
    }" > /dev/null

  curl -s -X POST "$API_BASE/consumptions" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"billId\":\"$BILL3\",
      \"userId\":\"$USER2\",
      \"units\":\"156.3\",
      \"meterValue\":\"23456.7\",
      \"recordedAt\":\"2025-02-28T20:05:00Z\"
    }" > /dev/null

  curl -s -X POST "$API_BASE/consumptions" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"billId\":\"$BILL3\",
      \"userId\":\"$USER3\",
      \"units\":\"98.4\",
      \"meterValue\":\"34567.8\",
      \"recordedAt\":\"2025-02-28T20:10:00Z\"
    }" > /dev/null

  echo "✓ Created 3 consumption readings"
fi

# Create loans
echo "9. Creating loans..."
if [ ! -z "$USER1" ] && [ ! -z "$USER2" ]; then
  curl -s -X POST "$API_BASE/loans" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"lenderId\":\"$USER1\",
      \"borrowerId\":\"$USER2\",
      \"amountPLN\":\"200.00\",
      \"note\":\"Pożyczka na zakupy\",
      \"dueDate\":\"2025-03-15T00:00:00Z\"
    }" > /dev/null

  curl -s -X POST "$API_BASE/loans" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"lenderId\":\"$USER2\",
      \"borrowerId\":\"$USER3\",
      \"amountPLN\":\"350.50\",
      \"note\":\"Pożyczka na naprawę samochodu\",
      \"dueDate\":\"2025-04-01T00:00:00Z\"
    }" > /dev/null

  echo "✓ Created 2 loans"
fi

# Create chores
echo "10. Creating chores..."
CHORE1=$(curl -s -X POST "$API_BASE/chores" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Sprzątanie kuchni",
    "description":"Mycie podłogi, wycieranie blatów, zmywanie naczyń",
    "frequency":"weekly",
    "difficulty":3,
    "priority":4,
    "assignmentMode":"round_robin",
    "notificationsEnabled":true,
    "reminderHours":24,
    "isActive":true
  }' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

CHORE2=$(curl -s -X POST "$API_BASE/chores" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Wynoszenie śmieci",
    "description":"Segregacja i wyniesienie śmieci do kontenerów",
    "frequency":"weekly",
    "difficulty":2,
    "priority":5,
    "assignmentMode":"round_robin",
    "notificationsEnabled":true,
    "reminderHours":12,
    "isActive":true
  }' | grep -o '"id":"[^"]*' | cut -d'"' -f4)

curl -s -X POST "$API_BASE/chores" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Pranie",
    "description":"Pranie ubrań i pościeli",
    "frequency":"weekly",
    "difficulty":2,
    "priority":3,
    "assignmentMode":"manual",
    "notificationsEnabled":false,
    "isActive":true
  }' > /dev/null

curl -s -X POST "$API_BASE/chores" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Sprzątanie łazienki",
    "description":"Mycie toalety, prysznica, luster",
    "frequency":"weekly",
    "difficulty":4,
    "priority":5,
    "assignmentMode":"round_robin",
    "notificationsEnabled":true,
    "reminderHours":24,
    "isActive":true
  }' > /dev/null

echo "✓ Created 4 chores"

# Create chore assignments
echo "11. Creating chore assignments..."
if [ ! -z "$CHORE1" ] && [ ! -z "$USER1" ]; then
  curl -s -X POST "$API_BASE/chore-assignments" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"choreId\":\"$CHORE1\",
      \"assigneeUserId\":\"$USER1\",
      \"dueDate\":\"2025-03-08T20:00:00Z\",
      \"status\":\"pending\"
    }" > /dev/null

  curl -s -X POST "$API_BASE/chore-assignments" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"choreId\":\"$CHORE2\",
      \"assigneeUserId\":\"$USER2\",
      \"dueDate\":\"2025-03-05T18:00:00Z\",
      \"status\":\"done\",
      \"completedAt\":\"2025-03-04T17:30:00Z\"
    }" > /dev/null

  echo "✓ Created 2 chore assignments"
fi

echo ""
echo "=== Seed Data Complete ==="
echo ""
echo "Created:"
echo "  - 2 groups (Anna i Piotr, Maria)"
echo "  - 3 users (Anna, Piotr, Maria)"
echo "  - 9 bills (3 electricity, 2 gas, 2 internet, 2 other)"
echo "  - 3 consumption readings"
echo "  - 2 loans"
echo "  - 4 chores"
echo "  - 2 chore assignments"
echo ""
echo "You can now:"
echo "  - Login at http://localhost:16161"
echo "  - Test with: anna@example.com / password123"
echo "  - Or admin: $ADMIN_EMAIL / $ADMIN_PASSWORD"
echo ""
