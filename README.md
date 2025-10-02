# Holy Home - Household Management Application

Personal, self-hosted household management for shared bills, utilities, loans, and chores.

## Project Status

### ✅ Completed Components

#### Infrastructure & Configuration
- [x] Project directory structure (backend/, frontend/, deploy/)
- [x] Docker Compose setup with 3 services (API, Frontend, MongoDB)
- [x] All Dockerfiles (Go API, Vue Frontend)
- [x] Environment configuration (`.env.example`)
- [x] Git ignore files

#### Backend (Go + Fiber)
- [x] **Core Setup**
  - Go module initialization
  - MongoDB connection with health checks and indexes
  - All 10 data models (Decimal128 for money/units)
  - Configuration management

- [x] **Authentication & Security**
  - JWT access & refresh tokens
  - Argon2id password hashing
  - TOTP 2FA with QR provisioning
  - Admin bootstrap from environment
  - Rate limiting for sensitive endpoints
  - Request ID tracking for structured logging
  - RBAC middleware (ADMIN, RESIDENT roles)

- [x] **API Endpoints**
  - **Auth**: `/auth/login`, `/auth/refresh`, `/auth/enable-2fa`, `/auth/disable-2fa`
  - **Users**: CRUD operations, password change, profile retrieval
  - **Groups**: Create, read, update, delete with weight management
  - **Bills**: Create, list, retrieve, allocate, post, close
  - **Consumptions**: Record readings, retrieve by bill/user
  - **Allocations**: View cost allocations
  - **Loans**: Create loans, record payments, calculate balances

- [x] **Business Logic**
  - Complex electricity allocation (personal usage + common area with weights)
  - Gas/Internet equal split
  - Shared budget allocation
  - Bill lifecycle (draft → posted → closed)
  - Loan tracking with partial repayments
  - Pairwise balance calculations

### 🚧 In Progress / TODO

#### Backend
- [ ] Chores API (create, assign, rotate, mark done)
- [ ] SSE endpoint for real-time events
- [ ] CSV/PDF export functionality

#### Frontend (Vue 3 + Vite)
- [ ] Project initialization (Vite, Vue 3, Pinia, Router)
- [ ] Tailwind CSS dark theme (purple #9333ea, pink #ec4899)
- [ ] Polish i18n (src/i18n/pl.json)
- [ ] **Views**:
  - Login & 2FA setup
  - Dashboard (balances, upcoming dates, chores)
  - Bills management
  - Readings input
  - Balance & Loans
  - Chores
  - Settings (users, groups)
- [ ] SSE client for real-time updates
- [ ] Export UI (CSV/PDF downloads)
- [ ] PWA configuration

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.25+ (for local development)
- Node.js current (for frontend development)

### Configuration

1. Copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

2. Generate admin password hash:
```bash
# Using any Argon2 tool
# The example hash is for "ChangeMe123!"
```

3. Update `.env` with real values:
   - `JWT_SECRET` and `JWT_REFRESH_SECRET` (use strong random strings)
   - `ADMIN_EMAIL` and `ADMIN_PASSWORD_HASH`

### Running with Docker Compose

```bash
cd deploy
docker-compose up -d
```

Services will be available at:
- API: http://localhost:16162
- Frontend: http://localhost:16161
- Health check: http://localhost:16162/healthz

### Local Development

#### Backend
```bash
cd backend
go mod tidy
go run ./cmd/api
```

#### Frontend
```bash
cd frontend
npm install
npm run dev
```

## API Documentation

### Authentication
- `POST /auth/login` - Login with email/password/TOTP
- `POST /auth/refresh` - Refresh access token
- `POST /auth/enable-2fa` - Enable TOTP 2FA
- `POST /auth/disable-2fa` - Disable TOTP 2FA

### Users & Groups
- `GET /users` - List all users [ADMIN]
- `POST /users` - Create user [ADMIN]
- `GET /users/me` - Get current user profile
- `GET /users/:id` - Get user by ID
- `PATCH /users/:id` - Update user [ADMIN]
- `POST /users/change-password` - Change own password
- `GET /groups` - List all groups
- `POST /groups` - Create group [ADMIN]
- `GET /groups/:id` - Get group by ID
- `PATCH /groups/:id` - Update group [ADMIN]
- `DELETE /groups/:id` - Delete group [ADMIN]

### Bills & Consumptions
- `POST /bills` - Create bill [ADMIN]
- `GET /bills?type=&from=&to=` - List bills with filters
- `GET /bills/:id` - Get bill by ID
- `POST /bills/:id/allocate` - Allocate costs [ADMIN]
- `POST /bills/:id/post` - Post bill (freeze allocations) [ADMIN]
- `POST /bills/:id/close` - Close bill (immutable) [ADMIN]
- `POST /consumptions` - Record consumption reading
- `GET /consumptions?billId=` - Get consumptions for bill
- `GET /allocations?billId=` - Get allocations for bill

### Loans & Balances
- `POST /loans` - Create loan
- `POST /loan-payments` - Record loan payment
- `GET /balances` - Get pairwise balances
- `GET /balances/me` - Get current user's balance
- `GET /balances/user/:id` - Get user's balance [ADMIN]

## Data Model

### Collections
- **users**: Email, password hash, role, group, TOTP secret, active status
- **groups**: Name, weight (default 1.0)
- **bills**: Type (electricity/gas/internet/shared), period, amount, units, status
- **consumptions**: Bill ID, user ID, units, meter value, recorded date
- **allocations**: Bill ID, subject (user/group), amount, units, method
- **payments**: Bill ID, payer, amount, paid date
- **loans**: Lender, borrower, amount, status (open/partial/settled)
- **loan_payments**: Loan ID, amount, paid date
- **chores**: Name
- **chore_assignments**: Chore ID, assignee, due date, status
- **notifications**: Channel (app), template, scheduled date, status

### Money & Units
- All monetary values use `Decimal128` (2 decimal places, banker's rounding)
- All units (kWh, m³) use `Decimal128` (3 decimal places)

## Allocation Rules

### Electricity
1. Personal usage cost = `user_units / sum_individual_units * cost_individual_pool`
2. Common area cost = `common_pool / sum_weights * user_weight`
3. Admin can override with custom weights
4. `posted` status freezes allocations
5. `closed` status makes bill immutable

### Gas / Internet / Shared
- Equal split among all active users/groups
- Optional per-usage for gas if readings exist

## Security

- Passwords: Argon2id (m=65536, t=3, p=1)
- JWT: HS256 with separate secrets for access/refresh tokens
- 2FA: TOTP (30s window, 6 digits, SHA1)
- Rate limits: 5 login attempts per 15 minutes
- RBAC: ADMIN (full access), RESIDENT (limited)

## Logging

Structured JSON logs (English) with:
- Timestamp
- Level (info, error, etc.)
- Service name
- Request ID
- User ID
- Route
- Latency (ms)
- Error details

## Next Steps

1. **Complete Chores API** - Task management and rotation
2. **Build Frontend** - Vue 3 application with Polish UI
3. **Add SSE Events** - Real-time updates for bills, chores
4. **Export Features** - CSV and PDF generation
5. **Testing** - Unit tests for allocation math, integration tests, E2E tests

## License

Private project - all rights reserved.