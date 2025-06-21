# Multi-stage build for Stavily Sensor Agent
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY shared/go.mod shared/go.sum ./shared/
COPY sensor-agent/go.mod sensor-agent/go.sum ./sensor-agent/

# Download dependencies
WORKDIR /app/shared
RUN go mod download

WORKDIR /app/sensor-agent
RUN go mod download

# Copy source code
COPY shared/ ../shared/
COPY sensor-agent/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev} -X main.buildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -o sensor-agent cmd/sensor-agent/main.go

# Final stage - minimal runtime image
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the binary
COPY --from=builder /app/sensor-agent/sensor-agent /usr/local/bin/sensor-agent

# Create non-root user
USER 65534:65534

# Create necessary directories
USER root
RUN mkdir -p /opt/stavily/plugins /opt/stavily/config /var/log/stavily \
    && chown -R 65534:65534 /opt/stavily /var/log/stavily
USER 65534:65534

# Set working directory
WORKDIR /opt/stavily

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/sensor-agent", "health"]

# Set default command
ENTRYPOINT ["/usr/local/bin/sensor-agent"]
CMD ["--config", "/opt/stavily/config/sensor-agent.yaml"] 