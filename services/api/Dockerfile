# Heroku-optimized Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Build the migration binary (requires migrate build tag)
RUN CGO_ENABLED=0 GOOS=linux go build -tags migrate -o migrate ./cmd/migrate

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the API binary and the migration binary
COPY --from=builder /app/main .
COPY --from=builder /app/migrate .

# Copy migration files so release_command can run ./migrate
COPY --from=builder /app/sql ./sql

# Create a non-root user and ensure it can run migrate and read sql/
RUN adduser -D appuser && chown -R appuser:appuser /app
USER appuser

# Expose port (Heroku will set PORT env var)
EXPOSE 8080

# Run the application
CMD ["./main"] 