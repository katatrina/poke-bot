# Build stage
FROM golang:1.25.1-alpine AS builder

# Thiết lập working directory
WORKDIR /build

# Copy go mod và sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o app ./cmd/server

# Run stage
FROM alpine:3.22

# Copy binary từ builder stage
COPY --from=builder /build/app /app

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app"]