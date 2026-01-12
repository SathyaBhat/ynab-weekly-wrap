# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app

# Stage 2: Runtime
FROM alpine:3.18

WORKDIR /app

# Install timezone data and curl for health checks
RUN apk add --no-cache tzdata curl

# Copy the binary from builder
COPY --from=builder /build/app .

# Copy config template
COPY configs/config.yaml.example configs/config.yaml

# Create a non-root user
RUN addgroup -g 1000 ynab && adduser -D -u 1000 -G ynab ynab
RUN chown -R ynab:ynab /app

USER ynab

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./app"]
