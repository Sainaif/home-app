# Deployment

## Quick Start

1. Edit `docker-compose.sqlite.yml` - set the 4 required values:
   - `JWT_SECRET` - generate with `openssl rand -base64 32`
   - `JWT_REFRESH_SECRET` - generate with `openssl rand -base64 32`
   - `ADMIN_EMAIL` - your admin email
   - `ADMIN_PASSWORD` - strong password (12+ chars)

2. Run:
   ```bash
   docker-compose -f docker-compose.sqlite.yml up -d
   ```

3. Access at http://localhost:16161

## Optional Configuration

Add any of these to the `environment:` section in docker-compose.sqlite.yml:

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
| `TZ` | UTC | Container timezone |

## Production Checklist

- [ ] Set `APP_DOMAIN` to your actual domain (required for WebAuthn/passkeys)
- [ ] Set `APP_BASE_URL` to your full URL (e.g., `https://home.yourdomain.com`)
- [ ] Set `ALLOWED_ORIGINS` to your domain (instead of `*`)
- [ ] Consider enabling `AUTH_2FA_ENABLED=true`
- [ ] Change admin password after first login

## Data

SQLite database is stored in the `holyhome_data` Docker volume at `/data/holyhome.db`.

To backup:
```bash
docker cp $(docker-compose -f docker-compose.sqlite.yml ps -q holyhome):/data/holyhome.db ./backup.db
```
