# Stage 1: Build
FROM golang:1.25.6-alpine3.23 AS builder

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
FROM alpine:3.23

WORKDIR /app

# Install timezone data
RUN apk add --no-cache tzdata

# Copy the binary from builder
COPY --from=builder /build/app .

# Create a non-root user
RUN addgroup -g 1000 ynab && adduser -D -u 1000 -G ynab ynab
RUN chown -R ynab:ynab /app

USER ynab

# Run the application
CMD ["./app"]
