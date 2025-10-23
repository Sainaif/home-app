# Holy Home - Household Management Application

A self-hosted household management application for tracking shared bills, utilities, loans, and chores. Built with Go, Vue 3, and MongoDB.

## Features

- **Multi-User Support**: User management with ADMIN and RESIDENT roles
- **Bill Management**: Track electricity, gas, internet, and custom bills
- **Smart Cost Allocation**: Complex electricity allocation with personal usage + common area pools
- **Consumption Tracking**: Record meter readings for accurate usage-based billing
- **Loan Management**: Track loans between users with partial repayment support
- **Balance Calculations**: Automatic pairwise debt netting across all users
- **Chores**: Task assignment and tracking (backend complete, frontend in progress)
- **Security**: JWT authentication, TOTP 2FA, Argon2id password hashing, RBAC
- **Dark Theme**: Modern purple/pink themed UI with Tailwind CSS
- **Polish Language**: Full Polish localization

## Project Status

### ✅ Completed

#### Infrastructure
- [x] Docker Compose setup (API, Frontend, MongoDB)
- [x] Environment configuration
- [x] Health check endpoints

#### Backend (Go + Fiber)
- [x] MongoDB connection with indexes and health checks
- [x] 10 data models with Decimal128 for money/units
- [x] JWT access (15m) & refresh tokens (720h)
- [x] TOTP 2FA with QR code provisioning
- [x] Rate limiting on sensitive endpoints
- [x] Request ID tracking and structured JSON logging
- [x] Complete API endpoints for:
  - Authentication (login, refresh, 2FA)
  - Users (CRUD, password management)
  - Groups (CRUD with weight management)
  - Bills (create, allocate, post, close)
  - Consumptions (meter readings)
  - Allocations (cost distributions)
  - Loans and loan payments
  - Balance calculations
  - Chores and chore assignments
- [x] Complex electricity allocation algorithm
- [x] Bill lifecycle management (draft → posted → closed)
- [x] Pairwise balance netting

#### Frontend (Vue 3 + Vite)
- [x] Project setup with Pinia, Vue Router, Vue I18n
- [x] Axios interceptors with JWT refresh
- [x] Dark theme with Tailwind CSS
- [x] Polish translations
- [x] All main views:
  - Login with 2FA support
  - Dashboard with balance overview
  - Bills management
  - Consumption readings
  - Balance & loans view
  - Settings (users, groups, profile)
- [x] Responsive components with Lucide icons
- [x] ECharts integration for data visualization

### 🚧 In Progress

#### Backend
- [ ] SSE endpoint for real-time events
- [ ] CSV/PDF export functionality

#### Frontend
- [ ] Chores view
- [ ] SSE client for real-time updates
- [ ] Export UI (CSV/PDF downloads)
- [ ] PWA configuration

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.25+ (for local development)
- Node.js 20+ (for frontend development)

### Running with Docker Compose (Recommended)

1. Copy environment file:
```bash
cp .env.example .env
```

2. Generate strong JWT secrets (optional, defaults provided):
```bash
openssl rand -base64 32  # For JWT_SECRET
openssl rand -base64 32  # For JWT_REFRESH_SECRET
```

3. Set admin credentials in `.env`:
```env
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=YourSecurePassword123!
```

4. Start all services:
```bash
cd deploy
docker-compose up -d
```

5. Access the application:
- **Frontend**: http://localhost:16161
- **API**: http://localhost:16162
- **Health check**: http://localhost:16162/healthz

6. Login with admin credentials from `.env`

### Local Development

#### Backend
```bash
cd backend
go mod tidy              # Install dependencies
go run ./cmd/api         # Run API (requires MongoDB and env vars)
go test ./...            # Run tests
go fmt ./...             # Format code
```

#### Frontend
```bash
cd frontend
npm install              # Install dependencies
npm run dev              # Dev server at http://localhost:5173
npm run build            # Build for production
npm run preview          # Preview production build
```

#### MongoDB Access
```bash
docker exec -it deploy-mongo-1 mongosh
# In mongosh:
use holyhome
db.users.find()
db.bills.find()
```

### Rebuilding After Changes
```bash
cd deploy
docker-compose build && docker-compose up -d
docker-compose logs -f api  # View API logs
```

## Architecture

### Technology Stack

**Backend**:
- Go 1.25+ with Fiber web framework
- MongoDB 8.0 for data persistence
- JWT authentication with refresh tokens
- TOTP for two-factor authentication

**Frontend**:
- Vue 3 with Composition API
- Vite for fast builds
- Pinia for state management
- Vue Router for navigation
- Vue I18n for internationalization
- Tailwind CSS for styling
- Axios for API calls
- ECharts for data visualization
- Lucide Vue for icons

### Backend Structure

```
backend/
├── cmd/api/main.go           # Application entry point
├── internal/
│   ├── config/               # Environment configuration
│   ├── database/             # MongoDB connection & indexing
│   ├── models/               # 10 data models with Decimal128
│   ├── handlers/             # HTTP request handlers
│   ├── services/             # Business logic layer
│   ├── middleware/           # Auth, RBAC, rate limiting
│   └── utils/                # JWT, crypto, decimal utilities
└── go.mod
```

### Frontend Structure

```
frontend/src/
├── main.js                   # App initialization
├── App.vue                   # Root component
├── router/index.js           # Route definitions
├── stores/                   # Pinia stores
├── views/                    # Page components
├── components/               # Reusable UI components
├── api/                      # API client
├── locales/pl.json           # Polish translations
└── composables/              # Composition functions
```

## API Documentation

Base URL: `http://localhost:16162` (Docker) or `http://localhost:3000` (local)

All endpoints require JWT bearer token except `/auth/login` and `/healthz`.

### Authentication (`/auth`)
- `POST /auth/login` - Login with email/password/TOTP (optional)
- `POST /auth/refresh` - Refresh access token using refresh token
- `POST /auth/enable-2fa` - Enable TOTP 2FA (returns QR code)
- `POST /auth/disable-2fa` - Disable TOTP 2FA

### Users (`/users`)
- `GET /users` - List all users **[ADMIN]**
- `POST /users` - Create new user **[ADMIN]**
- `GET /users/me` - Get current user profile
- `GET /users/:id` - Get user by ID
- `PATCH /users/:id` - Update user **[ADMIN]**
- `DELETE /users/:id` - Delete user **[ADMIN]**
- `POST /users/change-password` - Change own password

### Groups (`/groups`)
- `GET /groups` - List all groups
- `POST /groups` - Create group **[ADMIN]**
- `GET /groups/:id` - Get group by ID
- `PATCH /groups/:id` - Update group (name, weight) **[ADMIN]**
- `DELETE /groups/:id` - Delete group **[ADMIN]**

### Bills (`/bills`)
- `POST /bills` - Create bill **[ADMIN]**
- `GET /bills?type=&from=&to=&status=` - List bills with filters
- `GET /bills/:id` - Get bill details with allocations
- `POST /bills/:id/allocate` - Calculate cost allocations **[ADMIN]**
- `POST /bills/:id/post` - Post bill (freeze allocations) **[ADMIN]**
- `POST /bills/:id/close` - Close bill (make immutable) **[ADMIN]**

### Consumptions (`/consumptions`)
- `POST /consumptions` - Record meter reading
- `GET /consumptions?billId=&userId=` - Get consumptions by bill/user

### Allocations (`/allocations`)
- `GET /allocations?billId=&subjectId=` - Get cost allocations

### Payments (`/payments`)
- `POST /payments` - Record payment for bill
- `GET /payments?billId=` - Get payments for bill

### Loans (`/loans`, `/loan-payments`)
- `POST /loans` - Create loan between users
- `GET /loans?lenderId=&borrowerId=&status=` - List loans with filters
- `POST /loan-payments` - Record loan repayment
- `GET /loan-payments?loanId=` - Get payments for loan

### Balances (`/loans/balances`)
- `GET /loans/balances` - Get all pairwise balances **[ADMIN]**
- `GET /loans/balances/me` - Get current user's balances
- `GET /loans/balances/user/:id` - Get user's balances **[ADMIN]**

### Chores (`/chores`, `/chore-assignments`)
- `POST /chores` - Create chore **[ADMIN]**
- `GET /chores` - List all chores
- `POST /chore-assignments` - Assign chore to user **[ADMIN]**
- `GET /chore-assignments?assigneeId=&status=` - List assignments
- `PATCH /chore-assignments/:id` - Update assignment status

See [API_EXAMPLES.md](API_EXAMPLES.md) for detailed request/response examples.

## Data Model

MongoDB `holyhome` database with 10 collections:

### Users
- Email (unique)
- Password hash (Argon2id)
- Role (ADMIN, RESIDENT)
- Group reference (optional)
- TOTP secret (for 2FA)
- Active status

### Groups
- Name
- Weight (for cost allocation, default 1.0)

### Bills
- Type (electricity, gas, internet, inne/custom)
- Custom type (for "inne" category)
- Period (month/year)
- Total amount (Decimal128)
- Total units (Decimal128, optional)
- Status (draft, posted, closed)
- Created/updated timestamps

### Consumptions
- Bill reference
- User reference
- Units consumed (Decimal128)
- Meter value
- Recorded date

### Allocations
- Bill reference
- Subject (user or group ID)
- Subject type (user/group)
- Allocated amount (Decimal128)
- Allocated units (Decimal128, optional)
- Allocation method

### Payments
- Bill reference
- Payer reference
- Amount paid (Decimal128)
- Payment date

### Loans
- Lender reference
- Borrower reference
- Amount (Decimal128)
- Status (open, partial, settled)
- Created date

### Loan Payments
- Loan reference
- Amount (Decimal128)
- Payment date

### Chores
- Name
- Description

### Chore Assignments
- Chore reference
- Assignee reference
- Due date
- Status (pending, completed)

### Money & Units Precision
- **Monetary values**: `Decimal128` with 2 decimal places, banker's rounding (for PLN)
- **Utility units**: `Decimal128` with 3 decimal places (for kWh, m³)

## Business Logic

### Bill Lifecycle

1. **Draft**: Initial state, allocations can be recalculated
2. **Posted**: Allocations frozen, payments can be recorded
3. **Closed**: Immutable, bill is finalized

Transitions are one-way only: draft → posted → closed

### Cost Allocation

#### Electricity (Complex Algorithm)
The most sophisticated allocation in the system:

1. **Split pools**:
   - Personal pool: Based on individual meter readings
   - Common area pool: Shared across all residents/groups

2. **Calculate personal cost**:
   ```
   user_personal_cost = (user_units / sum_all_individual_units) × personal_pool
   ```

3. **Calculate common area cost**:
   ```
   user_common_cost = (common_pool / sum_of_weights) × user_or_group_weight
   ```

4. **Total allocation**:
   ```
   total_cost = user_personal_cost + user_common_cost
   ```

5. **Custom weights**: Admin can override default weights for fine-tuned allocation

Implementation: [backend/internal/services/bill_service.go](backend/internal/services/bill_service.go)

#### Gas / Internet / Custom Bills
- Equal split among all active users/groups
- Gas can optionally use per-usage if meter readings exist

#### Shared Budget ("Inne")
- Flexible custom bills for shared expenses
- Equal split by default

### Balance Calculations

Pairwise debt netting algorithm:

1. Aggregate all allocations (money owed)
2. Aggregate all payments (money paid)
3. Aggregate all loans (money borrowed/lent)
4. Calculate net balance between each user pair
5. Determine who owes whom and how much

Returns simplified debt graph for easy settlement.

Implementation: [backend/internal/services/loan_service.go](backend/internal/services/loan_service.go)

## Security

### Authentication & Authorization
- **Password hashing**: Argon2id (memory: 64MB, iterations: 3, parallelism: 1)
- **JWT tokens**: HS256 algorithm
  - Access token: 15 minutes lifetime
  - Refresh token: 720 hours (30 days) lifetime
  - Separate secrets for access/refresh
- **Two-Factor Authentication**: TOTP (30s window, 6 digits, SHA1)
- **Rate limiting**: 5 login attempts per 15 minutes per IP
- **RBAC**: Two roles
  - **ADMIN**: Full system access (user management, bill creation, allocations)
  - **RESIDENT**: Limited access (view bills, record consumptions, own profile)

### Security Best Practices
- All passwords stored as Argon2id hashes, never in plaintext
- JWT secrets must be strong random strings (use `openssl rand -base64 32`)
- Admin credentials provided via environment variables only
- HTTPS recommended for production deployments
- MongoDB authentication enabled in production

## Logging

Structured JSON logging with the following fields:
- `timestamp`: ISO 8601 format
- `level`: info, warn, error, debug
- `service`: "holy-home-api"
- `request_id`: Unique ID for request tracing
- `user_id`: Authenticated user (if applicable)
- `route`: HTTP endpoint
- `method`: HTTP method
- `status`: HTTP status code
- `latency_ms`: Request duration
- `error`: Error details (if applicable)
- `message`: Human-readable description

Example log entry:
```json
{
  "timestamp": "2025-01-15T10:30:45Z",
  "level": "info",
  "service": "holy-home-api",
  "request_id": "abc123",
  "user_id": "507f1f77bcf86cd799439011",
  "route": "/bills",
  "method": "GET",
  "status": 200,
  "latency_ms": 45,
  "message": "Bills retrieved successfully"
}
```

## Development Roadmap

### Immediate Next Steps
- [ ] Implement Chores view in frontend
- [ ] Add SSE endpoint for real-time updates
- [ ] Implement SSE client in frontend
- [ ] Add CSV/PDF export functionality
- [ ] PWA configuration for offline support

### Future Enhancements
- [ ] Recurring bills automation
- [ ] Email notifications for due bills
- [ ] Mobile app (React Native or Flutter)
- [ ] Multi-currency support
- [ ] Budget forecasting with ML
- [ ] Receipt image upload and OCR
- [ ] Telegram bot integration
- [ ] Multi-household support

### Testing
- [ ] Unit tests for allocation algorithms
- [ ] Integration tests for API endpoints
- [ ] E2E tests for critical user flows
- [ ] Load testing for production readiness

## Contributing

This is a personal project. If you'd like to contribute:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

Private project - all rights reserved.

## Support

For issues or questions:
- Create an issue in the repository
- Check [API_EXAMPLES.md](API_EXAMPLES.md) for usage examples