# ============================================================
# Stage 1: Build the Go binary
# ============================================================
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /wallet-api .

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /wallet-api /usr/local/bin/wallet-api

EXPOSE 8080

ENTRYPOINT ["wallet-api"]
