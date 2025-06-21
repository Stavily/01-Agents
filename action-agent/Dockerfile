# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
COPY ../shared/go.mod ../shared/go.sum ../shared/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .
COPY ../shared ../shared

# Build the binary
ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o action-agent \
    ./cmd/action-agent

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Create non-root user
RUN echo 'stavily:x:65534:65534:stavily:/:/sbin/nologin' >> /etc/passwd && \
    echo 'stavily:x:65534:' >> /etc/group

# Copy the binary
COPY --from=builder /build/action-agent /usr/local/bin/action-agent

# Create necessary directories
RUN mkdir -p /var/lib/stavily/action-agent \
             /tmp/stavily/action-agent \
             /var/cache/stavily/action-agent \
             /var/log/stavily \
             /etc/stavily

# Set ownership
RUN chown -R stavily:stavily /var/lib/stavily \
                             /tmp/stavily \
                             /var/cache/stavily \
                             /var/log/stavily \
                             /etc/stavily

# Switch to non-root user
USER stavily

# Set environment variables
ENV STAVILY_ACTION_CONFIG_FILE=/etc/stavily/action-agent.yaml
ENV STAVILY_ACTION_LOG_LEVEL=info

# Expose ports
EXPOSE 8081 9091

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/action-agent", "health"]

# Default command
ENTRYPOINT ["/usr/local/bin/action-agent"]
CMD ["--config", "/etc/stavily/action-agent.yaml"] 