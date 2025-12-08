# Holy Home

[![CI](https://github.com/Sainaif/home-app/actions/workflows/ci.yml/badge.svg)](https://github.com/Sainaif/home-app/actions/workflows/ci.yml)
[![Docker Publish](https://github.com/Sainaif/home-app/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/Sainaif/home-app/actions/workflows/docker-publish.yml)

**A self-hosted household management app for shared living situations.**

---

## What is Holy Home?

Living with roommates or family members often means dealing with shared expenses—utility bills, groceries, household supplies—and keeping track of who owes what can quickly become a headache. Holy Home solves this by providing a central place to:

- Record utility bills and automatically calculate each person's fair share
- Track meter readings so costs are split based on actual usage, not guesswork
- Manage informal loans between housemates ("I covered your share last month")
- Keep a running balance so everyone knows where they stand
- Coordinate household chores and shared supplies

Holy Home is designed to be **self-hosted**, meaning you run it on your own server or computer. Your financial data stays private and under your control.

---

## Features

### Bill Management
Track electricity, gas, internet, rent, and any custom bill types. Enter the total amount and let the app handle the math.

### Smart Cost Splitting
The app splits costs intelligently based on the bill type:
- **Metered utilities** (electricity): Personal usage from individual meters is charged directly. Common areas (hallway lights, shared appliances) are split equally.
- **Flat-rate bills** (internet, streaming): Split equally among all residents by default, or customize per bill.

### Meter Readings
Record consumption data from individual and shared meters. The app calculates each person's usage percentage for accurate billing.

### Loan Tracking
Keep track of money borrowed and lent between residents. "I paid for your groceries" or "You covered my rent" situations are logged and reflected in the balance.

### Balance Overview
A clear summary showing who owes money and who is owed. Settle up periodically or let balances carry forward.

### Household Supplies
Track shared purchases (toilet paper, cleaning supplies, etc.) and automatically add them to the cost-splitting system.

### Chore Management
Create and assign household tasks. Set up rotation schedules so chores are distributed fairly.

### Secure Authentication
Multiple login options: email, username, passkeys (WebAuthn), and optional two-factor authentication (TOTP).

---

## Quick Start

**Requirements:** Docker and Docker Compose

1. Edit `deploy/docker-compose.sqlite.yml` - set the 4 required values:
   - `JWT_SECRET` - generate with `openssl rand -base64 32`
   - `JWT_REFRESH_SECRET` - generate with `openssl rand -base64 32`
   - `ADMIN_EMAIL` - your admin email
   - `ADMIN_PASSWORD` - strong password (12+ chars)

2. Run:
   ```bash
   docker-compose -f deploy/docker-compose.sqlite.yml up -d
   ```

3. Access at **http://localhost:16161**

> The admin account is created automatically on first startup.

---

## Configuration

All configuration is done via environment variables in `docker-compose.sqlite.yml`.

### Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret for access tokens. Generate with: `openssl rand -base64 32` |
| `JWT_REFRESH_SECRET` | Secret for refresh tokens (use a different value) |
| `ADMIN_EMAIL` | Email for the initial admin account |
| `ADMIN_PASSWORD` | Password for the initial admin account |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_NAME` | Holy Home | Display name |
| `APP_ENV` | production | Environment mode (development/production) |
| `APP_DOMAIN` | localhost | Domain for WebAuthn passkeys |
| `APP_BASE_URL` | http://localhost:16161 | Full URL for generated links |
| `ALLOWED_ORIGINS` | * | CORS allowed origins |
| `JWT_ACCESS_TTL` | 15m | Access token lifetime |
| `JWT_REFRESH_TTL` | 720h | Refresh token lifetime (30 days) |
| `AUTH_2FA_ENABLED` | false | Enable TOTP two-factor auth |
| `AUTH_ALLOW_EMAIL_LOGIN` | true | Allow login with email |
| `AUTH_ALLOW_USERNAME_LOGIN` | false | Allow login with username |
| `LOG_LEVEL` | info | Logging level (debug/info/warn/error) |
| `LOG_FORMAT` | json | Log format (json/text) |
| `TZ` | Europe/Warsaw | Container timezone |
| `PUID` | (internal) | User ID for file ownership |
| `PGID` | (internal) | Group ID for file ownership |

---

## Volume Permissions

The container handles permissions automatically. Three options:

### Option 1: Named volume (default)
```yaml
volumes:
  - holyhome_data:/data
```
Works out of the box. Permissions handled internally.

### Option 2: Bind mount with PUID/PGID
```yaml
environment:
  PUID: 1000
  PGID: 1000
volumes:
  - ./data:/data
```
Files on host will be owned by the specified UID:GID. Find your IDs with `id -u` and `id -g`.

### Option 3: Bind mount with user directive
```yaml
user: "1000:1000"
volumes:
  - ./data:/data
```
Requires pre-creating the directory: `mkdir -p ./data`

---

## Data & Backups

SQLite database is stored at `/data/holyhome.db` inside the container.

- **Named volume**: Data in `holyhome_data` Docker volume
- **Bind mount**: Data in `./data/holyhome.db` on host

To backup:
```bash
# Named volume
docker cp $(docker-compose -f deploy/docker-compose.sqlite.yml ps -q holyhome):/data/holyhome.db ./backup.db

# Bind mount
cp ./data/holyhome.db ./backup.db
```

---

## Production Checklist

- [ ] Set `APP_DOMAIN` to your actual domain (required for WebAuthn/passkeys)
- [ ] Set `APP_BASE_URL` to your full URL (e.g., `https://home.yourdomain.com`)
- [ ] Set `ALLOWED_ORIGINS` to your domain (instead of `*`)
- [ ] Consider enabling `AUTH_2FA_ENABLED=true`
- [ ] Change admin password after first login

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Fiber v2 |
| Database | SQLite |
| Frontend | Vue 3, Vite, Tailwind CSS, Pinia |
| Auth | JWT, Argon2id, WebAuthn, TOTP |
| Deployment | Docker (single container) |

**Port:** 16161 (serves both frontend and API)

---

## Development

### Running Locally

```bash
# Backend
cd backend
go mod tidy
go run ./cmd/api

# Frontend (in a separate terminal)
cd frontend
npm install
npm run dev
```

### Running Tests

```bash
# Backend
cd backend
go test -v -race ./...

# Frontend
cd frontend
npm test
```

### Rebuilding Docker Images

```bash
docker-compose -f deploy/docker-compose.sqlite.yml up -d --build
```

---

## Project Structure

```
backend/
├── cmd/api/           # Application entry point
└── internal/
    ├── config/        # Environment configuration
    ├── database/      # SQLite connection
    ├── handlers/      # HTTP route handlers
    ├── middleware/    # Auth, rate limiting
    ├── models/        # Data structures
    ├── services/      # Business logic
    └── utils/         # JWT, passwords, TOTP, WebAuthn

frontend/
└── src/
    ├── api/           # HTTP client
    ├── components/    # Reusable UI components
    ├── composables/   # Vue 3 composition functions
    ├── locales/       # Translations
    ├── stores/        # Pinia state management
    └── views/         # Page components

deploy/                # Docker Compose configuration
```

---

## Security

- **Argon2id** password hashing with secure parameters
- **JWT tokens** with short-lived access (15 min) and long-lived refresh (30 days)
- **WebAuthn/Passkeys** for passwordless authentication
- **TOTP 2FA** for additional account protection
- **Rate limiting** on login attempts (5 per 15 minutes)

---

## License

This project is licensed under the **Creative Commons Attribution-NonCommercial 4.0 International License (CC BY-NC 4.0)**.

### You are free to:

- **Use** — run the software for personal, family, or internal household use
- **Share** — copy and redistribute the software in any medium or format
- **Adapt** — remix, transform, and build upon the software

### Under the following terms:

- **Attribution** — You must give appropriate credit, provide a link to the license, and indicate if changes were made
- **NonCommercial** — You may not use the software for commercial purposes. This means you cannot sell the software, offer it as a paid service, or use it in a business context for profit

### What this means in practice:

- Running Holy Home for your household or shared living situation
- Sharing the code with friends who want to use it for their own homes
- Modifying the code and sharing your improvements
- Hosting it on a home server or VPS for personal use

### What is not permitted:

- Selling Holy Home or derivative works
- Offering Holy Home as a paid hosted service
- Using Holy Home as part of a commercial property management business

For the full license text, see [CC BY-NC 4.0](https://creativecommons.org/licenses/by-nc/4.0/).

---

## Contributing

Contributions are welcome! Please open an issue to discuss proposed changes before submitting a pull request.
