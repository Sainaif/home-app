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

```bash
# Clone the repository
git clone https://github.com/Sainaif/home-app.git
cd home-app

# Configure environment
cp .env.example .env
# Edit .env with your settings (see Configuration below)

# Start the application
cd deploy
docker-compose up -d
```

Open **http://localhost:16161** in your browser and log in with your admin credentials.

> The admin account is created automatically on first startup using the credentials from your `.env` file.

---

## Configuration

Copy `.env.example` to `.env` and set the following:

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
| `APP_ENV` | `development` | Set to `production` for production deployments |
| `APP_DOMAIN` | `localhost` | Your domain (required for passkey authentication) |
| `AUTH_ALLOW_USERNAME_LOGIN` | `false` | Allow login with username instead of email |
| `AUTH_2FA_ENABLED` | `false` | Enable two-factor authentication |

See `.env.example` for the complete list of options.

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24, Fiber v2 |
| Database | MongoDB 8.0 |
| Frontend | Vue 3, Vite, Tailwind CSS, Pinia |
| Auth | JWT, Argon2id, WebAuthn, TOTP |
| Deployment | Docker, Docker Compose |

**Ports:** Frontend on `16161`, API on `16162`, MongoDB on `27017`

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
# Backend (requires MongoDB replica set)
cd backend
go test -v -race ./...

# Frontend
cd frontend
npm test
```

### Rebuilding Docker Images

```bash
cd deploy
docker-compose build && docker-compose up -d
```

---

## Project Structure

```
backend/
├── cmd/api/           # Application entry point
└── internal/
    ├── config/        # Environment configuration
    ├── database/      # MongoDB connection
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
