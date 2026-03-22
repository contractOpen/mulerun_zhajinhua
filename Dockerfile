# ==============================================================================
# Stage 1: Build frontends (Vue 3 + Vite)
# ==============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy both frontend projects (frontend-tg needs @shared alias to ../frontend/src/composables)
COPY frontend/ frontend/
COPY frontend-tg/ frontend-tg/

# Build main frontend (te and pe modes)
WORKDIR /build/frontend
RUN npm install && \
    npm run build:te && \
    npm run build:pe

# Build TG Mini App frontend
WORKDIR /build/frontend-tg
RUN npm install && \
    npm run build

# ==============================================================================
# Stage 2: Build Go backend (CGO required for mattn/go-sqlite3)
# ==============================================================================
FROM golang:1.21-bookworm AS backend-builder

WORKDIR /build/backend

# Download dependencies first (layer caching)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy backend source
COPY backend/ .

# Copy built static files from frontend stage
COPY --from=frontend-builder /build/backend/static-te ./static-te/
COPY --from=frontend-builder /build/backend/static-pe ./static-pe/
COPY --from=frontend-builder /build/backend/static-tg ./static-tg/

# Build with CGO enabled (required for go-sqlite3)
ENV CGO_ENABLED=1
RUN go build -o zhajinhua .

# ==============================================================================
# Stage 3: Runtime
# ==============================================================================
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=backend-builder /build/backend/zhajinhua .

# Copy static files
COPY --from=backend-builder /build/backend/static-te ./static-te/
COPY --from=backend-builder /build/backend/static-pe ./static-pe/
COPY --from=backend-builder /build/backend/static-tg ./static-tg/

# Create data directory for SQLite persistence
RUN mkdir -p /app/data

# Default environment
ENV APP_MODE=te
ENV PORT=8080
ENV ADMIN_PASSWORD=admin123

EXPOSE 8080

CMD ["./zhajinhua"]
