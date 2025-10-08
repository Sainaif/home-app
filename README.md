# Holy Home

Self-hosted app for managing household bills, utilities, and loans. Built for shared living situations.

## What it does

- Track bills (electricity, gas, internet, custom)
- Split costs automatically based on usage
- Record meter readings
- Track loans between people
- See who owes what
- Dark mode UI in Polish

## Tech Stack

- Backend: Go + MongoDB
- Frontend: Vue 3 + Tailwind
- Docker for easy deployment

## Status

Most things work. Still need to add:
- Chores management (backend done)
- Real-time updates
- Export to PDF/CSV

## Getting Started

You need Docker installed.

1. Copy `.env.example` to `.env` and set your admin email/password
2. Run it:
```bash
cd deploy
docker-compose up -d
```
3. Open http://localhost:16161 and login

That's it. The app creates the admin user automatically on first start.

### If you want to develop locally

Backend:
```bash
cd backend
go mod tidy
go run ./cmd/api
```

Frontend:
```bash
cd frontend
npm install
npm run dev
```

### Rebuild after changes
```bash
cd deploy
docker-compose build && docker-compose up -d
```

## How it works

### Bill allocation

The app splits electricity bills in a smart way:
- Personal usage (from your meter) gets charged to you
- Common areas (hallway, kitchen) split equally
- Gas/internet split equally by default

Everything else is straightforward - track what you owe, what you paid, and who owes you.

## API

API runs on `http://localhost:16162`

## Database

Uses MongoDB with these main collections:
- users (email, password, role)
- groups (name, weight for splitting costs)
- bills (type, amount, period, status)
- consumptions (meter readings)
- allocations (who owes what)
- payments (who paid what)
- loans (money borrowed/lent)

## Security

- Passwords hashed with Argon2id
- JWT tokens for auth (15min access + 30 day refresh)
- Optional 2FA with TOTP
- Rate limiting on login (5 attempts per 15min)