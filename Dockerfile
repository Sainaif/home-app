# syntax=docker/dockerfile:1

# Holy Home v1.5 - Single Container Build
# Combines Vue.js frontend + Go backend + SQLite into one image

# =============================================================================
# Stage 1: Build Frontend
# =============================================================================
FROM node:20-alpine AS frontend

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci --silent

COPY frontend/ ./
RUN npm run build

# =============================================================================
# Stage 2: Build Backend (with CGO for SQLite)
# =============================================================================
FROM golang:1.24-alpine AS backend

# Install build dependencies for SQLite
RUN apk add --no-cache git ca-certificates tzdata build-base sqlite-dev

WORKDIR /backend

# Download Go dependencies first (for cache)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ ./

# Copy built frontend to static directory
# Note: Vite outputs to ../backend/internal/static/dist relative to frontend/
COPY --from=frontend /backend/internal/static/dist ./internal/static/dist/

# Build with CGO enabled for SQLite support
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w -linkmode external -extldflags '-static'" \
    -o /holyhome ./cmd/api

# =============================================================================
# Stage 3: Runtime Image
# =============================================================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite \
    sqlite-libs \
    su-exec \
    && addgroup -S app \
    && adduser -S app -G app \
    && mkdir -p /data \
    && chown app:app /data

WORKDIR /app

# Copy the compiled binary and entrypoint
COPY --from=backend /holyhome .
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set default environment variables
ENV TZ=Europe/Warsaw \
    DATABASE_PATH=/data/holyhome.db \
    APP_PORT=3000 \
    APP_HOST=0.0.0.0

# Expose the application port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:3000/healthz || exit 1

# Data volume for SQLite database
VOLUME ["/data"]

# Start the application (entrypoint handles permissions and drops to app user)
ENTRYPOINT ["/entrypoint.sh"]
