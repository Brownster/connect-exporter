# Multi-stage build for Go exporter
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o aws-connect-exporter ./cmd/connect-exporter_exporter

# Final stage - minimal runtime image
FROM scratch

# Import certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
WORKDIR /app
COPY --from=builder /app/aws-connect-exporter /aws-connect-exporter
COPY config.aws-connect.yml /app/config.yml

# Create non-root user (if needed for advanced security)
# COPY --from=builder /etc/passwd /etc/passwd
# USER nobody

# Expose port
EXPOSE 9090

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:9090/health || exit 1

# Command to run
ENTRYPOINT ["/aws-connect-exporter"]

# Default arguments
CMD ["--config.file=/app/config.yml"]

# Labels for better maintainability
LABEL maintainer="your-email@company.com"
LABEL description="Prometheus exporter for AWS Connect"
LABEL version="1.0.0"
