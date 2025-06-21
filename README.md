# Stavily Agents

This directory contains the implementation of Stavily's two-agent architecture: **Sensor Agents** and **Action Agents**. Both agents are lightweight, compiled Go binaries designed for minimal resource consumption, efficiency, and security.

## Architecture Overview

### Sensor Agents
- **Purpose**: Monitor systems and other things like apis and detect trigger conditions (python scripts)
- **Communication**: Report to orchestrator at `agents.stavily.com` via secure API
- **Deployment**: Customer infrastructure (Docker/VM/Kubernetes)
- **Permissions**: Read-only system access

### Action Agents  
- **Purpose**: Execute automation tasks based on workflow definitions
- **Communication**: Poll orchestrator for action requests via secure API
- **Deployment**: Customer infrastructure (Docker/VM/Kubernetes)
- **Permissions**: Execution capabilities with sandboxed plugin environment

## Directory Structure

```
agents/
├── shared/                     # Shared libraries and utilities
├── sensor-agent/              # Sensor agent implementation
├── action-agent/             # Action agent implementation
└── scripts/                  # Build and deployment scripts
```

## Quick Start

### Prerequisites
- Go 1.21+
- Docker (optional)
- Make

### Build All Agents
```bash
make build
```

### Run Locally
```bash
# Sensor Agent
./bin/sensor-agent --config configs/dev-sensor.yaml

# Action Agent  
./bin/action-agent --config configs/dev-action.yaml
```

### Docker Deployment
```bash
# Build images
make docker-build

# Run core agents only
docker-compose up -d

# For development environment
docker-compose -f deployments/docker/docker-compose.dev.yml up -d
```

## Development

### Testing
```bash
make test
```

### Linting
```bash
make lint
```

### Documentation
```bash
make docs
```

## Security

- All agents use mTLS for API communication
- Sandboxed plugin execution environment
- Non-root user execution in all deployment models
- Strict tenant isolation and scoping

## Plugin Architecture

Both agents support hot-reloadable plugins:
- **Sensor Agent**: Trigger detection plugins (Python SDK)
- **Action Agent**: Action and output execution plugins (Python SDK)

## Observability

- Structured logging with configurable levels
- Health check endpoints
- Distributed tracing support

## Deployment Options

1. **Bare Metal**: Systemd service or background process
2. **Docker**: Container deployment with minimal base images  
3. **Kubernetes**: DaemonSet for cluster-wide deployment

## Configuration

Both agents use YAML configuration files with environment variable overrides. See individual agent directories for specific configuration options.

## Contributing

Please see the main project [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines and coding standards.

## License

See [LICENSE](../LICENSE) for license information. 

## Key Features

- **Lightweight**: Minimal resource footprint (~10-50MB RAM)
- **Secure**: mTLS communication, sandboxed execution, non-root users
- **Plugin-oriented**: Hot-reloadable plugins for extensibility
- **Multi-tenant**: Support for multiple organizations/tenants
- **Cross-platform**: Linux, macOS, Windows support
- **Deployment flexibility**: Bare metal, Docker, Kubernetes

## Agent Directory Structure and Naming

Starting with this version, Stavily agents use **agent-specific directories** and **systemd service names** to support multiple agents of the same type on a single machine:

- **Base Directory**: `agent-{AGENT_ID}` instead of `.stavily`
- **Service Names**: `sensor-agent-{AGENT_ID}.service` and `action-agent-{AGENT_ID}.service`
- **Multiple Agents**: You can run multiple sensor or action agents with different IDs

Example structure:
```
/var/lib/stavily/
├── agent-sensor-web-01/        # First sensor agent
│   ├── config/
│   ├── data/
│   └── logs/
├── agent-sensor-db-01/         # Second sensor agent  
│   ├── config/
│   ├── data/
│   └── logs/
└── agent-action-exec-01/       # Action agent
    ├── config/
    ├── data/
    └── logs/
```

**SystemD Service Examples:**
- `/etc/systemd/system/sensor-agent-web-01.service`
- `/etc/systemd/system/sensor-agent-db-01.service`
- `/etc/systemd/system/action-agent-exec-01.service`

## Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- Docker (for containerized deployment)
- Valid Stavily account and API credentials

### Build All Agents

```bash
# Build both agents for current platform
make build

# Build for specific platform
make build-linux
make build-windows
make build-darwin

# Build Docker images
make docker-build

# Cross-platform build (all supported platforms)
make build-all
```

### Deploy Sensor Agent (Docker)

```bash
# Create configuration directory
mkdir -p ~/agent-{AGENT_ID}/sensor-agent

# Create configuration file
cat > ~/agent-{AGENT_ID}/sensor-agent/config.yaml << EOF
agent:
  id: "sensor-001"
  name: "My Sensor Agent"
  type: "sensor"
  organization_id: "your-org-id"

api:
  base_url: "https://agents.stavily.com"
  auth:
    type: "certificate"
    cert_file: "/app/agent-{AGENT_ID}/certs/client.crt"
    key_file: "/app/agent-{AGENT_ID}/certs/client.key"
    ca_file: "/app/agent-{AGENT_ID}/certs/ca.crt"

logging:
  level: "info"
  format: "json"
EOF

# Run with Docker
docker run -d \
  --name stavily-sensor \
  --restart unless-stopped \
  -v ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw \
  -v /var/log:/var/log:ro \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  stavily/sensor-agent:latest
```

### Deploy Action Agent (Docker)

```bash
# Create configuration directory
mkdir -p ~/agent-{AGENT_ID}/action-agent

# Create configuration file
cat > ~/agent-{AGENT_ID}/action-agent/config.yaml << EOF
agent:
  id: "action-001"
  name: "My Action Agent"
  type: "action"
  organization_id: "your-org-id"

api:
  base_url: "https://agents.stavily.com"
  auth:
    type: "certificate"
    cert_file: "/app/agent-{AGENT_ID}/certs/client.crt"
    key_file: "/app/agent-{AGENT_ID}/certs/client.key"
    ca_file: "/app/agent-{AGENT_ID}/certs/ca.crt"

logging:
  level: "info"
  format: "json"
EOF

# Run with Docker
docker run -d \
  --name stavily-action \
  --restart unless-stopped \
  -v ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  stavily/action-agent:latest
```

## Building from Source

### Build Requirements

- Go 1.21 or later
- Make (for using Makefile)
- Docker (for containerized builds)

### Build Commands

```bash
# Install dependencies
make deps

# Run tests
make test

# Build all agents
make build

# Build specific agent
make build-sensor
make build-action

# Build for all platforms
make build-all

# Create release packages
make package
```

### Build Artifacts

Built binaries are placed in the `bin/` directory:

```
bin/
├── sensor-agent-linux-amd64
├── sensor-agent-linux-arm64
├── sensor-agent-darwin-amd64
├── sensor-agent-darwin-arm64
├── sensor-agent-windows-amd64.exe
├── action-agent-linux-amd64
├── action-agent-linux-arm64
├── action-agent-darwin-amd64
├── action-agent-darwin-arm64
└── action-agent-windows-amd64.exe
```

## Deployment Methods

### 1. Bare Metal Deployment

#### System Requirements

- **CPU**: 1 core minimum, 2+ cores recommended
- **Memory**: 512MB minimum, 1GB+ recommended
- **Disk**: 100MB for binaries, 1GB+ for logs and data
- **Network**: HTTPS outbound access to `agents.stavily.com`
- **OS**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+), macOS 11+, Windows 10+

#### Installation Steps

1. **Download and Install Binary**

```bash
# Download latest release
curl -L https://github.com/stavily/agents/releases/latest/download/sensor-agent-linux-amd64 -o sensor-agent
chmod +x sensor-agent
sudo mv sensor-agent /usr/local/bin/

# Or for Action Agent
curl -L https://github.com/stavily/agents/releases/latest/download/action-agent-linux-amd64 -o action-agent
chmod +x action-agent
sudo mv action-agent /usr/local/bin/
```

2. **Create System User**

```bash
# Create dedicated user for security
sudo useradd --system --shell /bin/false --home-dir /var/lib/stavily stavily
sudo mkdir -p /var/lib/stavily/
sudo chown -R stavily:stavily /var/lib/stavily
```

3. **Configure Base Directory**

The agents use a base directory (`agent-{AGENT_ID}` by default) for all configuration, data, and runtime files:

```bash
# Default structure
/var/lib/stavily/agent-{AGENT_ID}/
├── config/
│   ├── agent.yaml          # Main configuration
│   ├── plugins/            # Plugin configurations
│   └── certificates/       # TLS certificates
├── data/
│   ├── plugins/            # Plugin binaries and data
│   ├── cache/              # Temporary cache files
│   └── state/              # Agent state files
├── logs/
│   ├── agent.log           # Agent logs
│   ├── plugins/            # Plugin logs
│   └── audit/              # Audit logs
└── tmp/                    # Temporary files
```

4. **Create Configuration**

```bash
# Create configuration directory
sudo mkdir -p /var/lib/stavily/agent-{AGENT_ID}/config
sudo mkdir -p /var/lib/stavily/agent-{AGENT_ID}/certificates

# Create main configuration file
sudo tee /var/lib/stavily/agent-{AGENT_ID}/config/agent.yaml << EOF
agent:
  id: "sensor-$(hostname)-$(date +%s)"
  name: "Sensor Agent - $(hostname)"
  type: "sensor"
  organization_id: "your-org-id"
  base_dir: "/var/lib/stavily/agent-{AGENT_ID}"

api:
  base_url: "https://agents.stavily.com"
  timeout: "30s"
  auth:
    type: "certificate"
    cert_file: "/var/lib/stavily/agent-{AGENT_ID}/certificates/client.crt"
    key_file: "/var/lib/stavily/agent-{AGENT_ID}/certificates/client.key"
    ca_file: "/var/lib/stavily/agent-{AGENT_ID}/certificates/ca.crt"

logging:
  level: "info"
  format: "json"
  file: "/var/lib/stavily/agent-{AGENT_ID}/logs/agent.log"
  max_size: 100
  max_backups: 5
  max_age: 30

security:
  sandbox:
    enabled: true
    user: "stavily"
    chroot: "/var/lib/stavily"
  tls:
    enabled: true
    min_version: "1.3"

plugins:
  dir: "/var/lib/stavily/agent-{AGENT_ID}/data/plugins"
  config_dir: "/var/lib/stavily/agent-{AGENT_ID}/config/plugins"
  auto_update: true
  max_memory: "256MB"
  timeout: "5m"

health:
  port: 8080
  enabled: true
EOF

# Set proper ownership
sudo chown -R stavily:stavily /var/lib/stavily/agent-{AGENT_ID}
sudo chmod 600 /var/lib/stavily/agent-{AGENT_ID}/config/agent.yaml
```

5. **Install Certificates**

```bash
# Download and install certificates from Stavily
# (Replace with actual certificate provisioning process)
sudo curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  https://api.stavily.com/v1/agents/certificates/client.crt \
  -o /var/lib/stavily/agent-{AGENT_ID}/certificates/client.crt

sudo curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  https://api.stavily.com/v1/agents/certificates/client.key \
  -o /var/lib/stavily/agent-{AGENT_ID}/certificates/client.key

sudo curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  https://api.stavily.com/v1/agents/certificates/ca.crt \
  -o /var/lib/stavily/agent-{AGENT_ID}/certificates/ca.crt

# Set proper permissions
sudo chmod 600 /var/lib/stavily/agent-{AGENT_ID}/certificates/*
sudo chown stavily:stavily /var/lib/stavily/agent-{AGENT_ID}/certificates/*
```

6. **Create SystemD Service**

```bash
# Create service file
sudo tee /etc/systemd/system/sensor-agent-{AGENT_ID}.service << EOF
[Unit]
Description=Stavily Sensor Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=stavily
Group=stavily
ExecStart=/usr/local/bin/sensor-agent --config=/var/lib/stavily/agent-{AGENT_ID}/config/agent.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=stavily-sensor
KillMode=mixed
KillSignal=SIGTERM

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/stavily
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable sensor-agent-{AGENT_ID}.service
sudo systemctl start sensor-agent-{AGENT_ID}.service

# Check status
sudo systemctl status sensor-agent-{AGENT_ID}.service
```

#### Custom Base Directory

To use a custom base directory instead of `agent-{AGENT_ID}`:

```bash
# Set custom base directory
export STAVILY_BASE_DIR="/opt/mycompany/stavily"

# Create directory structure
sudo mkdir -p $STAVILY_BASE_DIR/{config,data,logs,tmp}
sudo chown -R stavily:stavily $STAVILY_BASE_DIR

# Update configuration
sudo tee $STAVILY_BASE_DIR/config/agent.yaml << EOF
agent:
  base_dir: "$STAVILY_BASE_DIR"
  # ... rest of configuration
EOF

# Update systemd service
sudo sed -i "s|/var/lib/stavily/agent-{AGENT_ID}|$STAVILY_BASE_DIR|g" /etc/systemd/system/sensor-agent-{AGENT_ID}.service
sudo systemctl daemon-reload
sudo systemctl restart sensor-agent-{AGENT_ID}.service
```

### 2. Docker Deployment

#### Docker Requirements

- Docker 20.10+
- Docker Compose 2.0+ (optional)
- 1GB+ available disk space
- Network access to `agents.stavily.com`

#### Basic Docker Deployment

1. **Prepare Host Directory**

```bash
# Create base directory on host
mkdir -p ~/agent-{AGENT_ID}/{config,data,logs,certificates}

# Set proper permissions
chmod 755 ~/agent-{AGENT_ID}
chmod 700 ~/agent-{AGENT_ID}/certificates
```

2. **Create Configuration**

```bash
# Create agent configuration
cat > ~/agent-{AGENT_ID}/config/agent.yaml << EOF
agent:
  id: "sensor-docker-$(hostname)-$(date +%s)"
  name: "Docker Sensor Agent - $(hostname)"
  type: "sensor"
  organization_id: "your-org-id"
  base_dir: "/app/agent-{AGENT_ID}"

api:
  base_url: "https://agents.stavily.com"
  timeout: "30s"
  auth:
    type: "certificate"
    cert_file: "/app/agent-{AGENT_ID}/certificates/client.crt"
    key_file: "/app/agent-{AGENT_ID}/certificates/client.key"
    ca_file: "/app/agent-{AGENT_ID}/certificates/ca.crt"

logging:
  level: "info"
  format: "json"
  file: "/app/agent-{AGENT_ID}/logs/agent.log"

plugins:
  dir: "/app/agent-{AGENT_ID}/data/plugins"
  config_dir: "/app/agent-{AGENT_ID}/config/plugins"

health:
  port: 8080
  enabled: true
EOF
```

3. **Install Certificates**

```bash
# Download certificates (replace with actual provisioning)
curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  https://api.stavily.com/v1/agents/certificates/bundle.tar.gz \
  -o ~/agent-{AGENT_ID}/certificates/bundle.tar.gz

# Extract certificates
cd ~/agent-{AGENT_ID}/certificates
tar -xzf bundle.tar.gz
rm bundle.tar.gz
```

4. **Run Sensor Agent Container**

```bash
docker run -d \
  --name stavily-sensor \
  --restart unless-stopped \
  --network host \
  -v ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw \
  -v /var/log:/host/var/log:ro \
  -v /proc:/host/proc:ro \
  -v /sys:/host/sys:ro \
  -v /etc:/host/etc:ro \
  --security-opt no-new-privileges:true \
  --user 1000:1000 \
  stavily/sensor-agent:latest \
  --config=/app/agent-{AGENT_ID}/config/agent.yaml
```

5. **Run Action Agent Container**

```bash
docker run -d \
  --name stavily-action \
  --restart unless-stopped \
  --network host \
  -v ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp:/tmp \
  --security-opt no-new-privileges:true \
  --user 1000:1000 \
  stavily/action-agent:latest \
  --config=/app/agent-{AGENT_ID}/config/agent.yaml
```

#### Docker Compose Deployment

1. **Create Docker Compose File**

```bash
cat > docker-compose.yml << EOF
version: '3.8'

services:
  sensor-agent:
    image: stavily/sensor-agent:latest
    container_name: stavily-sensor
    restart: unless-stopped
    network_mode: host
    user: "1000:1000"
    security_opt:
      - no-new-privileges:true
    volumes:
      - ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw
      - /var/log:/host/var/log:ro
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /etc:/host/etc:ro
    command: ["--config=/app/agent-{AGENT_ID}/config/sensor.yaml"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"

  action-agent:
    image: stavily/action-agent:latest
    container_name: stavily-action
    restart: unless-stopped
    network_mode: host
    user: "1000:1000"
    security_opt:
      - no-new-privileges:true
    volumes:
      - ~/agent-{AGENT_ID}:/app/agent-{AGENT_ID}:rw
      - /var/run/docker.sock:/var/run/docker.sock
      - /tmp:/tmp
    command: ["--config=/app/agent-{AGENT_ID}/config/action.yaml"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
EOF
```

2. **Deploy with Docker Compose**

```bash
# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f sensor-agent
docker-compose logs -f action-agent

# Stop services
docker-compose down
```

#### Custom Volume Mounts

For different base directories or custom mount points:

```bash
# Using custom base directory
export STAVILY_BASE_DIR="/opt/stavily"
mkdir -p $STAVILY_BASE_DIR

# Run with custom mount
docker run -d \
  --name stavily-sensor \
  -v $STAVILY_BASE_DIR:/app/agent-{AGENT_ID}:rw \
  -e STAVILY_BASE_DIR=/app/agent-{AGENT_ID} \
  stavily/sensor-agent:latest

# Or with Docker Compose override
cat > docker-compose.override.yml << EOF
version: '3.8'
services:
  sensor-agent:
    volumes:
      - /opt/stavily:/app/agent-{AGENT_ID}:rw
    environment:
      - STAVILY_BASE_DIR=/app/agent-{AGENT_ID}
EOF
```

### 3. Kubernetes Deployment

#### Prerequisites

- Kubernetes 1.20+
- kubectl configured
- Persistent storage available

#### Basic Kubernetes Deployment

1. **Create Namespace**

```bash
kubectl create namespace stavily-agents
```

2. **Create ConfigMap**

```bash
kubectl create configmap sensor-agent-config \
  --from-file=agent.yaml=~/agent-{AGENT_ID}/config/agent.yaml \
  -n stavily-agents
```

3. **Create Secret for Certificates**

```bash
kubectl create secret generic agent-certificates \
  --from-file=client.crt=~/agent-{AGENT_ID}/certificates/client.crt \
  --from-file=client.key=~/agent-{AGENT_ID}/certificates/client.key \
  --from-file=ca.crt=~/agent-{AGENT_ID}/certificates/ca.crt \
  -n stavily-agents
```

4. **Create PersistentVolumeClaim**

```bash
cat > pvc.yaml << EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: stavily-data
  namespace: stavily-agents
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
EOF

kubectl apply -f pvc.yaml
```

5. **Deploy Sensor Agent**

```bash
cat > sensor-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sensor-agent
  namespace: stavily-agents
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sensor-agent
  template:
    metadata:
      labels:
        app: sensor-agent
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
      - name: sensor-agent
        image: stavily/sensor-agent:latest
        args: ["--config=/app/agent-{AGENT_ID}/config/agent.yaml"]
        ports:
        - containerPort: 8080
          name: health
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        volumeMounts:
        - name: config
          mountPath: /app/agent-{AGENT_ID}/config
          readOnly: true
        - name: certificates
          mountPath: /app/agent-{AGENT_ID}/certificates
          readOnly: true
        - name: data
          mountPath: /app/agent-{AGENT_ID}/data
        - name: logs
          mountPath: /app/agent-{AGENT_ID}/logs
        - name: host-proc
          mountPath: /host/proc
          readOnly: true
        - name: host-sys
          mountPath: /host/sys
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health
            port: health
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: sensor-agent-config
      - name: certificates
        secret:
          secretName: agent-certificates
      - name: data
        persistentVolumeClaim:
          claimName: stavily-data
      - name: logs
        emptyDir: {}
      - name: host-proc
        hostPath:
          path: /proc
      - name: host-sys
        hostPath:
          path: /sys
EOF

kubectl apply -f sensor-deployment.yaml
```

## Configuration Reference

### Agent Configuration

The agent configuration file (`agent.yaml`) supports the following sections:

#### Agent Settings

```yaml
agent:
  id: "unique-agent-id"                    # Required: Unique agent identifier
  name: "My Agent"                         # Required: Human-readable name
  type: "sensor|action"                    # Required: Agent type
  organization_id: "org-123"               # Required: Organization ID
  base_dir: "/path/to/agent-{AGENT_ID}"   # Optional: Base directory (default: agent-{AGENT_ID})
  environment: "production"                # Optional: Environment tag
  tags:                                    # Optional: Custom tags
    - "datacenter:us-east-1"
    - "role:sensor"
  heartbeat_interval: "30s"               # Optional: Heartbeat frequency
  registration_retry_interval: "60s"       # Optional: Registration retry interval
```

#### API Configuration

```yaml
api:
  base_url: "https://agents.stavily.com"  # Required: Orchestrator URL
  timeout: "30s"                          # Optional: Request timeout
  retry:
    max_attempts: 3                       # Optional: Max retry attempts
    backoff: "exponential"                # Optional: Backoff strategy
    initial_interval: "1s"                # Optional: Initial retry interval
    max_interval: "60s"                   # Optional: Maximum retry interval
  auth:
    type: "certificate|jwt"               # Required: Authentication type
    # Certificate auth
    cert_file: "/path/to/client.crt"      # Required for certificate auth
    key_file: "/path/to/client.key"       # Required for certificate auth
    ca_file: "/path/to/ca.crt"           # Required for certificate auth
    # JWT auth
    token_file: "/path/to/token"          # Required for JWT auth
    refresh_threshold: "5m"               # Optional: Token refresh threshold
  rate_limit:
    requests_per_second: 10               # Optional: Rate limit
    burst: 20                             # Optional: Burst capacity
```

#### Logging Configuration

```yaml
logging:
  level: "debug|info|warn|error"          # Optional: Log level (default: info)
  format: "json|text"                     # Optional: Log format (default: json)
  file: "/path/to/agent.log"              # Optional: Log file path
  max_size: 100                           # Optional: Max log file size (MB)
  max_backups: 5                          # Optional: Max backup files
  max_age: 30                             # Optional: Max age (days)
  compress: true                          # Optional: Compress old logs
```

#### Security Configuration

```yaml
security:
  sandbox:
    enabled: true                         # Optional: Enable sandboxing
    user: "stavily"                       # Optional: Sandbox user
    chroot: "/var/lib/stavily"           # Optional: Chroot directory
  tls:
    enabled: true                         # Optional: Enable TLS
    min_version: "1.3"                    # Optional: Minimum TLS version
    cipher_suites: []                     # Optional: Allowed cipher suites
  audit:
    enabled: true                         # Optional: Enable audit logging
    file: "/path/to/audit.log"           # Optional: Audit log file
```

#### Plugin Configuration

```yaml
plugins:
  dir: "/path/to/plugins"                 # Optional: Plugin directory
  config_dir: "/path/to/plugin-configs"   # Optional: Plugin config directory
  auto_update: true                       # Optional: Auto-update plugins
  update_interval: "1h"                   # Optional: Update check interval
  max_memory: "256MB"                     # Optional: Max plugin memory
  timeout: "5m"                          # Optional: Plugin execution timeout
  allowed_plugins: []                     # Optional: Whitelist of allowed plugins
  blocked_plugins: []                     # Optional: Blacklist of blocked plugins
```

#### Health Check Configuration

```yaml
health:
  enabled: true                           # Optional: Enable health endpoint
  port: 8080                             # Optional: Health check port
  path: "/health"                        # Optional: Health check path
  bind: "0.0.0.0"                       # Optional: Bind address
```

#### Metrics Configuration

```yaml
metrics:
  enabled: true                           # Optional: Enable metrics
  port: 9090                             # Optional: Metrics port
  path: "/metrics"                       # Optional: Metrics path
  bind: "127.0.0.1"                     # Optional: Bind address
  namespace: "stavily_agent"             # Optional: Metrics namespace
```

### Environment Variables

All configuration values can be overridden using environment variables with the `STAVILY_` prefix:

```bash
# Agent settings
export STAVILY_AGENT_ID="sensor-001"
export STAVILY_AGENT_NAME="My Sensor Agent"
export STAVILY_AGENT_TYPE="sensor"
export STAVILY_AGENT_ORGANIZATION_ID="org-123"
export STAVILY_AGENT_BASE_DIR="/opt/stavily"

# API settings
export STAVILY_API_BASE_URL="https://agents.stavily.com"
export STAVILY_API_TIMEOUT="30s"
export STAVILY_API_AUTH_TYPE="certificate"
export STAVILY_API_AUTH_CERT_FILE="/certs/client.crt"
export STAVILY_API_AUTH_KEY_FILE="/certs/client.key"
export STAVILY_API_AUTH_CA_FILE="/certs/ca.crt"

# Logging settings
export STAVILY_LOGGING_LEVEL="info"
export STAVILY_LOGGING_FORMAT="json"
export STAVILY_LOGGING_FILE="/var/log/stavily/agent.log"

# Security settings
export STAVILY_SECURITY_SANDBOX_ENABLED="true"
export STAVILY_SECURITY_TLS_ENABLED="true"

# Plugin settings
export STAVILY_PLUGINS_DIR="/opt/stavily/plugins"
export STAVILY_PLUGINS_AUTO_UPDATE="true"

# Health check settings
export STAVILY_HEALTH_ENABLED="true"
export STAVILY_HEALTH_PORT="8080"

# Metrics settings
export STAVILY_METRICS_ENABLED="true"
export STAVILY_METRICS_PORT="9090"
```

## Troubleshooting

### Health Checks

Both agents expose health check endpoints:

```bash
# Check agent health
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready

# Check metrics
curl http://localhost:9090/metrics
```

### Log Analysis

```bash
# View real-time logs (bare metal)
sudo journalctl -u stavily-sensor -f

# View real-time logs (Docker)
docker logs -f stavily-sensor

# View real-time logs (Kubernetes)
kubectl logs -f deployment/sensor-agent -n stavily-agents

# Search for errors
grep -i error ~/agent-{AGENT_ID}/logs/agent.log

# Monitor plugin activity
tail -f ~/agent-{AGENT_ID}/logs/plugins/*.log
```

### Common Issues

1. **Agent Registration Fails**
   - Check network connectivity to `agents.stavily.com`
   - Verify certificates are valid and not expired
   - Ensure organization ID is correct

2. **Plugin Load Failures**
   - Check plugin directory permissions
   - Verify plugin compatibility
   - Review plugin-specific logs

3. **High Memory Usage**
   - Review plugin memory limits
   - Check for memory leaks in custom plugins
   - Monitor system resources

4. **Certificate Expiration**
   - Set up certificate auto-renewal
   - Monitor certificate expiration dates
   - Implement alerting for certificate issues

### Performance Tuning

```yaml
# High-performance configuration
api:
  timeout: "10s"
  retry:
    max_attempts: 5
    initial_interval: "500ms"

plugins:
  max_memory: "1GB"
  timeout: "10m"

logging:
  level: "warn"  # Reduce log verbosity
  max_size: 500
  compress: true

metrics:
  enabled: true  # Enable health checks
```

## Security Best Practices

### Certificate Management

1. **Use Strong Certificates**
   - RSA 4096-bit or ECDSA P-384
   - Valid certificate chain
   - Regular rotation (90 days recommended)

2. **Secure Storage**
   - Certificates should be readable only by agent user
   - Use hardware security modules (HSM) when available
   - Encrypt certificate private keys

### Network Security

1. **Firewall Configuration**
   - Allow outbound HTTPS to `agents.stavily.com`
   - Block unnecessary inbound connections
   - Use network segmentation

2. **TLS Configuration**
   - Use TLS 1.3 minimum
   - Disable weak cipher suites
   - Enable certificate pinning

### System Security

1. **User Permissions**
   - Run agents as non-root user
   - Use minimal required permissions
   - Enable sandboxing when possible

2. **File System Security**
   - Secure base directory permissions (700)
   - Regular security updates
   - Monitor file integrity

## Support and Documentation

- **Documentation**: https://docs.stavily.com/agents
- **API Reference**: https://api.stavily.com/docs
- **Support**: support@stavily.com
- **Community**: https://community.stavily.com
- **Issues**: https://github.com/stavily/agents/issues 