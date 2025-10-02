# Holy Home API Examples

Complete examples for testing all endpoints.

## Setup

```bash
# Start services
cd deploy && docker-compose up -d

# Wait for services to be ready
sleep 10

# Set base URL
API_URL="http://localhost:8080"
```

## 1. Authentication

### Login
```bash
curl -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.pl",
    "password": "ChangeMe123!"
  }'
```

Save the returned `access` token:
```bash
TOKEN="eyJhbGc..."
```

### Refresh Token
```bash
REFRESH_TOKEN="..."

curl -X POST $API_URL/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "'$REFRESH_TOKEN'"
  }'
```

### Enable 2FA
```bash
curl -X POST $API_URL/auth/enable-2fa \
  -H "Authorization: Bearer $TOKEN"
```

## 2. Users & Groups

### Create User
```bash
curl -X POST $API_URL/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user1@example.pl",
    "role": "RESIDENT",
    "tempPassword": "TempPass123!"
  }'
```

### Get All Users
```bash
curl $API_URL/users \
  -H "Authorization: Bearer $TOKEN"
```

### Create Group
```bash
curl -X POST $API_URL/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Couple 1",
    "weight": 2.0
  }'
```

### Get Groups
```bash
curl $API_URL/groups \
  -H "Authorization: Bearer $TOKEN"
```

## 3. Bills & Allocations

### Create Electricity Bill
```bash
curl -X POST $API_URL/bills \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "electricity",
    "periodStart": "2025-09-01T00:00:00Z",
    "periodEnd": "2025-09-30T23:59:59Z",
    "totalAmountPLN": 450.00,
    "totalUnits": 300.0,
    "notes": "September 2025"
  }'
```

Save bill ID:
```bash
BILL_ID="66f9..."
```

### Record Consumption
```bash
USER_ID="66f8..."

curl -X POST $API_URL/consumptions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "billId": "'$BILL_ID'",
    "userId": "'$USER_ID'",
    "units": 150.5,
    "meterValue": 12345.5,
    "recordedAt": "2025-09-30T12:00:00Z"
  }'
```

### Allocate Bill
```bash
curl -X POST $API_URL/bills/$BILL_ID/allocate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "proportional"
  }'
```

### Post Bill (Freeze Allocations)
```bash
curl -X POST $API_URL/bills/$BILL_ID/post \
  -H "Authorization: Bearer $TOKEN"
```

### Close Bill (Make Immutable)
```bash
curl -X POST $API_URL/bills/$BILL_ID/close \
  -H "Authorization: Bearer $TOKEN"
```

### Get Bills with Filters
```bash
# All bills
curl "$API_URL/bills" \
  -H "Authorization: Bearer $TOKEN"

# Electricity bills only
curl "$API_URL/bills?type=electricity" \
  -H "Authorization: Bearer $TOKEN"

# Bills in date range
curl "$API_URL/bills?from=2025-09-01T00:00:00Z&to=2025-09-30T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN"
```

### Get Allocations
```bash
curl "$API_URL/allocations?billId=$BILL_ID" \
  -H "Authorization: Bearer $TOKEN"
```

## 4. Loans & Balances

### Create Loan
```bash
LENDER_ID="66f8..."
BORROWER_ID="66f9..."

curl -X POST $API_URL/loans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lenderId": "'$LENDER_ID'",
    "borrowerId": "'$BORROWER_ID'",
    "amountPLN": 500.00,
    "note": "Rent advance"
  }'
```

### Record Loan Payment
```bash
LOAN_ID="66fa..."

curl -X POST $API_URL/loan-payments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "loanId": "'$LOAN_ID'",
    "amountPLN": 200.00,
    "paidAt": "2025-10-01T12:00:00Z",
    "note": "First installment"
  }'
```

### Get All Balances
```bash
curl $API_URL/loans/balances \
  -H "Authorization: Bearer $TOKEN"
```

### Get My Balance
```bash
curl $API_URL/loans/balances/me \
  -H "Authorization: Bearer $TOKEN"
```

## 5. Chores

### Create Chore
```bash
curl -X POST $API_URL/chores \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Take out trash"
  }'
```

### Assign Chore
```bash
CHORE_ID="66fb..."
ASSIGNEE_ID="66f8..."

curl -X POST $API_URL/chores/assign \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "choreId": "'$CHORE_ID'",
    "assigneeUserId": "'$ASSIGNEE_ID'",
    "dueDate": "2025-10-01T00:00:00Z"
  }'
```

### Rotate Chore (Auto-assign Next User)
```bash
curl -X POST $API_URL/chores/$CHORE_ID/rotate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "dueDate": "2025-10-08T00:00:00Z"
  }'
```

### Mark Chore Done
```bash
ASSIGNMENT_ID="66fc..."

curl -X PATCH $API_URL/chore-assignments/$ASSIGNMENT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "done"
  }'
```

### Get My Chores
```bash
# All my chores
curl "$API_URL/chore-assignments/me" \
  -H "Authorization: Bearer $TOKEN"

# Only pending
curl "$API_URL/chore-assignments/me?status=pending" \
  -H "Authorization: Bearer $TOKEN"
```

### Swap Chore Assignments
```bash
ASSIGN1_ID="66fc..."
ASSIGN2_ID="66fd..."

curl -X POST $API_URL/chores/swap \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "assignment1Id": "'$ASSIGN1_ID'",
    "assignment2Id": "'$ASSIGN2_ID'"
  }'
```

## 6. Server-Sent Events (SSE)

### Connect to Event Stream
```bash
# Open connection and listen for events
curl -N $API_URL/events/stream \
  -H "Authorization: Bearer $TOKEN"
```

You'll receive:
```
data: {"type":"connected","timestamp":"2025-09-29T..."}

: heartbeat

event: bill.created
data: {"id":"...","type":"bill.created","data":{...},"timestamp":"..."}
```

### Test Event Broadcasting (From Another Terminal)
```bash
# Create a bill - should trigger bill.created event
curl -X POST $API_URL/bills \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "gas",
    "periodStart": "2025-10-01T00:00:00Z",
    "periodEnd": "2025-10-31T23:59:59Z",
    "totalAmountPLN": 120.00
  }'
```

## 7. Health Check

```bash
# API health
curl $API_URL/healthz
```

## Complete Workflow Example

```bash
# 1. Login
TOKEN=$(curl -s -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.pl","password":"ChangeMe123!"}' \
  | jq -r '.access')

# 2. Create users
USER1_ID=$(curl -s -X POST $API_URL/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user1@example.pl","role":"RESIDENT"}' \
  | jq -r '.id')

USER2_ID=$(curl -s -X POST $API_URL/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"user2@example.pl","role":"RESIDENT"}' \
  | jq -r '.id')

# 3. Create electricity bill
BILL_ID=$(curl -s -X POST $API_URL/bills \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"electricity",
    "periodStart":"2025-09-01T00:00:00Z",
    "periodEnd":"2025-09-30T23:59:59Z",
    "totalAmountPLN":450.00,
    "totalUnits":300.0
  }' | jq -r '.id')

# 4. Record consumptions
curl -s -X POST $API_URL/consumptions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"billId\":\"$BILL_ID\",\"userId\":\"$USER1_ID\",\"units\":150.0,\"recordedAt\":\"2025-09-30T12:00:00Z\"}"

curl -s -X POST $API_URL/consumptions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"billId\":\"$BILL_ID\",\"userId\":\"$USER2_ID\",\"units\":120.0,\"recordedAt\":\"2025-09-30T12:00:00Z\"}"

# 5. Allocate costs
curl -s -X POST $API_URL/bills/$BILL_ID/allocate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"strategy":"proportional"}'

# 6. View allocations
curl -s "$API_URL/allocations?billId=$BILL_ID" \
  -H "Authorization: Bearer $TOKEN" | jq

echo "✅ Complete workflow executed successfully!"
```

## Notes

- Replace `$TOKEN`, `$BILL_ID`, `$USER_ID`, etc. with actual IDs from responses
- Use `jq` for pretty-printing JSON responses
- Add `-v` flag to curl for verbose output (headers, etc.)
- For SSE, use `-N` flag to disable buffering

## Troubleshooting

### 401 Unauthorized
- Check token is valid: `echo $TOKEN`
- Token may have expired (15 min default), get a new one

### 403 Forbidden
- Check user role (some endpoints require ADMIN)
- Login as admin for admin-only endpoints

### 400 Bad Request
- Check request body JSON syntax
- Verify all required fields are present
- Check date formats (ISO 8601: `YYYY-MM-DDTHH:mm:ssZ`)

### Connection Refused
- Ensure services are running: `docker-compose ps`
- Check logs: `docker-compose logs api`