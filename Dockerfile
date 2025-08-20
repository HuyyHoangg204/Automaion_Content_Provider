# Build stage
FROM golang:1.23.0-alpine AS builder

# Architecture argument
ARG TARGETARCH
ARG TARGETOS

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy application source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o green-provider-services-backend ./cmd/server

# Production stage
FROM alpine:3.21.3

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=builder /app/green-provider-services-backend .
COPY --from=builder /app/docs ./docs

# Expose the application's port
EXPOSE 8080

# Run the application
CMD ["./green-provider-services-backend"]
