# ==============================================================================
# Stage 1: Build Go backend
# ==============================================================================
FROM golang:1.23-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o zhajinhua .

# ==============================================================================
# Stage 2: Runtime
# ==============================================================================
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl sqlite3 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /build/zhajinhua ./zhajinhua
COPY --from=builder /build/static ./static
COPY --from=builder /build/static-te ./static-te
COPY --from=builder /build/static-pe ./static-pe
COPY --from=builder /build/static-tg ./static-tg

RUN mkdir -p /app/data

ENV APP_MODE=pe
ENV PORT=8080
ENV DB_PATH=/app/data/zhajinhua.db
ENV ADMIN_PASSWORD=admin123

EXPOSE 8080

CMD ["./zhajinhua"]
