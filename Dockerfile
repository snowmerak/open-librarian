# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for Go modules
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy swagger files
COPY --from=builder /app/cmd/server/swagger ./cmd/server/swagger/

# Copy public files
COPY --from=builder /app/cmd/server/public ./cmd/server/public/

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
