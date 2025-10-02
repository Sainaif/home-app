# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Holy Home is a self-hosted household management application for tracking shared bills, utilities, loans, and chores. It uses a microservices architecture with:
- **Backend**: Go 1.25+ with Fiber framework
- **Frontend**: Vue 3 with Vite, Pinia, Vue Router, and Tailwind CSS
- **Database**: MongoDB 8.0

## Development Commands

### Backend (Go)
```bash
cd backend
go mod tidy              # Install/update dependencies
go run ./cmd/api         # Run locally (requires MongoDB and env vars)
go build ./cmd/api       # Build binary
go test ./...            # Run tests
go fmt ./...             # Format code
```

### Frontend (Vue)
```bash
cd frontend
npm install              # Install dependencies
npm run dev              # Run dev server (port 5173 for local dev)
npm run build            # Build for production
npm run preview          # Preview production build
```

**Note:** When running via Docker Compose, the frontend is served on port 16161.

### Docker Compose (Recommended)
```bash
cd deploy
docker-compose up -d                      # Start all services
docker-compose logs -f api                # View API logs
docker-compose down                       # Stop services
docker-compose build && docker-compose up -d  # Rebuild after code changes
```

### MongoDB Access
```bash
docker exec -it deploy-mongo-1 mongosh
# In mongosh:
use holyhome
db.users.find()
db.bills.find()
```

## Architecture

### Backend Structure (`/backend`)

The Go backend uses Fiber framework with clean architecture principles:

- **`cmd/api/main.go`**: Application entry point, initializes services and routes
- **`internal/config/`**: Environment configuration management
- **`internal/database/`**: MongoDB connection with health checks and indexing
- **`internal/models/`**: 10 data models (User, Group, Bill, Consumption, Allocation, Payment, Loan, LoanPayment, Chore, ChoreAssignment, Notification)
- **`internal/handlers/`**: HTTP request handlers for all endpoints
- **`internal/services/`**: Business logic layer (auth, bills, loans, balance calculations)
- **`internal/middleware/`**: Auth (JWT), RBAC, request ID tracking, rate limiting
- **`internal/utils/`**: JWT, Argon2id crypto, Decimal128 rounding utilities

**Key Backend Concepts:**

1. **Authentication**: JWT access tokens (15m) + refresh tokens (720h), TOTP 2FA support
2. **Authorization**: RBAC with ADMIN and RESIDENT roles
3. **Money & Units**: All monetary values use `primitive.Decimal128` with banker's rounding (2dp for PLN, 3dp for units)
4. **Bill Lifecycle**: `draft` → `posted` (frozen allocations) → `closed` (immutable)
5. **Electricity Allocation**: Complex cost distribution with personal usage pool + common area pool weighted by group size
6. **Balance Calculations**: Automatic pairwise debt netting between users

### Frontend Structure (`/frontend/src`)

Vue 3 SPA with composition API:

- **`main.js`**: App initialization with Pinia, Router, i18n, axios interceptors
- **`App.vue`**: Root component with navigation and dark theme (purple/pink)
- **`router/index.js`**: 7 routes (Login, Dashboard, Bills, Readings, Balance, Chores, Settings)
- **`stores/`**: Pinia stores for auth and application state
- **`views/`**: Page components
- **`components/`**: Reusable UI components
- **`api/`**: API client with axios
- **`locales/pl.json`**: Polish translations
- **`composables/`**: Shared composition functions

**Frontend Stack**: Vue 3, Pinia, Vue Router, Vue I18n, Axios, Tailwind CSS, ECharts, Lucide Vue icons

## Environment Configuration

Required environment variables (see `.env.example`):
- `APP_PORT=3000` - Backend API port
- `MONGO_URI=mongodb://mongo:27017` - MongoDB connection string
- `MONGO_DB=holyhome` - Database name
- `JWT_SECRET` and `JWT_REFRESH_SECRET` - Must be strong random strings
- `ADMIN_EMAIL` and `ADMIN_PASSWORD` - Bootstrap admin credentials

## API Structure

Base URL: `http://localhost:16162` (Docker) or `http://localhost:3000` (local dev)

**Authentication** (`/auth`): login, refresh, enable-2fa, disable-2fa
**Users** (`/users`): CRUD operations, password management
**Groups** (`/groups`): CRUD with weight management
**Bills** (`/bills`): Create, list, allocate, post, close
**Consumptions** (`/consumptions`): Record meter readings
**Allocations** (`/allocations`): View cost distributions
**Loans** (`/loans`, `/loan-payments`): Create loans and track repayments
**Balances** (`/loans/balances`): Pairwise balance calculations
**Chores** (`/chores`, `/chore-assignments`): Task management

All API endpoints require JWT bearer token except login. ADMIN role required for user/group/bill management.

## Database Collections

MongoDB `holyhome` database with 10 collections:
- **users**: Email, password hash, role, group reference, TOTP secret
- **groups**: Name, weight (for cost allocation)
- **bills**: Type (electricity/gas/internet/inne), custom_type (for "inne"), period, amount, units, status
- **consumptions**: Bill reference, user reference, meter readings
- **allocations**: Bill reference, subject (user/group), allocated costs
- **payments**: Bill reference, payer, amount
- **loans**: Lender, borrower, amount, status (open/partial/settled)
- **loan_payments**: Loan reference, repayment amount
- **chores**: Name, description
- **chore_assignments**: Chore reference, assignee, due date, status
- **notifications**: Channel (app), template, scheduled date, status

## Key Business Logic

### Electricity Cost Allocation

The most complex allocation logic in the system:

1. Split total cost into personal pool (based on individual meter readings) and common area pool
2. Personal cost = `user_units / sum_all_individual_units * personal_pool`
3. Common area cost = `common_pool / sum_of_weights * user_or_group_weight`
4. Admin can provide custom weights to override default equal split

Implementation: `backend/internal/services/bill_service.go`

### Balance Calculations

Pairwise debt netting across all users:
- Aggregates allocations, payments, and loans
- Calculates net balances between each user pair
- Returns who owes whom and how much

Implementation: `backend/internal/services/loan_service.go`

## Testing

Health check endpoint:
- API: `http://localhost:16162/healthz` (Docker) or `http://localhost:3000/healthz` (local dev)

See [API_EXAMPLES.md](API_EXAMPLES.md) for detailed API usage examples.

## Important Notes

- All monetary values use Decimal128, not floats - use `utils.TwoDecimal()` for PLN amounts
- Bill status transitions are one-way: draft → posted → closed (no reverting)
- Admin bootstrap happens automatically on first startup using env vars
- Rate limiting is enabled for login endpoint (5 attempts per 15 minutes)
- Frontend uses Polish language (pl.json) as primary locale
- All structured logs are in JSON format with request ID tracking
