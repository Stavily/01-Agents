# Stavily Agents Configuration Guide

This guide provides comprehensive documentation for configuring Stavily agents, including all available options, environment variables, and advanced configuration scenarios.

## Table of Contents

1. [Configuration File Structure](#configuration-file-structure)
2. [Base Directory Configuration](#base-directory-configuration)
3. [Agent-Specific Configuration](#agent-specific-configuration)
4. [Environment Variables](#environment-variables)
5. [Advanced Configuration Scenarios](#advanced-configuration-scenarios)
6. [Configuration Validation](#configuration-validation)

## Configuration File Structure

Stavily agents use YAML configuration files with the following top-level sections:

```yaml
# agent.yaml - Complete configuration example
agent:          # Agent identification and basic settings
api:            # API communication settings
logging:        # Logging configuration
security:       # Security and encryption settings
plugins:        # Plugin management settings
health:         # Health check configuration
metrics:        # Metrics collection settings
```

## Base Directory Configuration

### Default Base Directory Structure

The agents use a base directory (`agent-{AGENT_ID}` by default) for all configuration, data, and runtime files:

```
agent-{AGENT_ID}/                   # Base directory (configurable)
├── config/                         # Configuration files
│   ├── agent.yaml                  # Main agent configuration
│   ├── plugins/                    # Plugin-specific configurations
│   │   ├── prometheus-trigger.yaml
│   │   ├── email-output.yaml
│   │   └── shell-action.yaml
│   └── certificates/               # TLS certificates
│       ├── client.crt
│       ├── client.key
│       └── ca.crt
├── data/                          # Persistent data
│   ├── plugins/                   # Plugin binaries and data
│   │   ├── prometheus-trigger.so
│   │   ├── email-output.so
│   │   └── shell-action.so
│   ├── cache/                     # Temporary cache files
│   │   ├── api-cache.db
│   │   └── plugin-cache/
│   └── state/                     # Agent state files
│       ├── agent-state.json
│       ├── plugin-state/
│       └── workflow-state/
├── logs/                          # Log files
│   ├── agent.log                  # Main agent log
│   ├── plugins/                   # Plugin-specific logs
│   │   ├── prometheus-trigger.log
│   │   └── email-output.log
│   └── audit/                     # Audit logs
│       ├── security.log
│       └── compliance.log
└── tmp/                           # Temporary files
    ├── downloads/
    ├── sandbox/
    └── workdir/
```

### Custom Base Directory

To use a custom base directory:

```yaml
# Method 1: Configuration file
agent:
  base_dir: "/opt/mycompany/stavily"

# Method 2: Environment variable
export STAVILY_BASE_DIR="/opt/mycompany/stavily"

# Method 3: Command line flag
sensor-agent --base-dir="/opt/mycompany/stavily"
```

### Directory Permissions

Recommended permissions for production:

```bash
# Base directory: readable by agent user only
chmod 700 /opt/stavily/agent-{AGENT_ID}

# Configuration: readable by agent user only
chmod 600 /opt/stavily/agent-{AGENT_ID}/config/agent.yaml
chmod 700 /opt/stavily/agent-{AGENT_ID}/config/plugins/

# Certificates: readable by agent user only
chmod 600 /opt/stavily/agent-{AGENT_ID}/config/certificates/*

# Data directory: writable by agent user
chmod 755 /opt/stavily/agent-{AGENT_ID}/data/
chmod 755 /opt/stavily/agent-{AGENT_ID}/data/plugins/

# Logs: writable by agent user
chmod 755 /opt/stavily/agent-{AGENT_ID}/logs/

# Temporary: writable by agent user
chmod 755 /opt/stavily/agent-{AGENT_ID}/tmp/
```

## Agent-Specific Configuration

### Sensor Agent Configuration

```yaml
# sensor-agent.yaml - Standard configuration for all sensor agents
agent:
  id: "sensor-{HOSTNAME}-001"
  name: "Sensor Agent - {HOSTNAME}"
  type: "sensor"
  organization_id: "your-org-id"
  base_dir: "/opt/stavily/agent-{AGENT_ID}"
  environment: "production"
  tags:
    - "datacenter:us-east-1"
    - "tier:monitoring"
    - "role:sensor"
  heartbeat_interval: "30s"
  registration_retry_interval: "60s"

api:
  base_url: "https://agents.stavily.com"
  timeout: "30s"
  retry:
    max_attempts: 3
    backoff: "exponential"
    initial_interval: "1s"
    max_interval: "60s"
  auth:
    type: "certificate"
    cert_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/client.crt"
    key_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/client.key"
    ca_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/ca.crt"
  rate_limit:
    requests_per_second: 10
    burst: 20

logging:
  level: "info"
  format: "json"
  file: "/opt/stavily/agent-{AGENT_ID}/logs/sensor-agent.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true

security:
  sandbox:
    enabled: true
    user: "stavily"
    chroot: "/opt/stavily"
  tls:
    enabled: true
    min_version: "1.3"
    cipher_suites: []
  audit:
    enabled: true
    file: "/opt/stavily/agent-{AGENT_ID}/logs/audit/sensor-audit.log"

plugins:
  dir: "/opt/stavily/agent-{AGENT_ID}/data/plugins"
  config_dir: "/opt/stavily/agent-{AGENT_ID}/config/plugins"
  auto_update: true
  update_interval: "1h"
  max_memory: "256MB"
  timeout: "5m"
  allowed_plugins:
    - "cpu-monitor"
    - "disk-monitor"
    - "memory-monitor"
    - "file-watcher"
  blocked_plugins: []

health:
  enabled: true
  port: 8080
  path: "/health"
  bind: "0.0.0.0"

metrics:
  enabled: true
  port: 9090
  path: "/metrics"
  bind: "127.0.0.1"
  namespace: "stavily_sensor"

# Sensor-specific settings
sensor:
  trigger_check_interval: "10s"
  max_concurrent_triggers: 5
  trigger_timeout: "30s"
  event_buffer_size: 1000
  batch_size: 10
  flush_interval: "5s"
```

### Action Agent Configuration

```yaml
# action-agent.yaml
agent:
  id: "action-prod-exec-01"
  name: "Production Action Executor"
  type: "action"
  organization_id: "org-12345"
  base_dir: "/opt/stavily/agent-{AGENT_ID}"
  environment: "production"
  tags:
    - "datacenter:us-east-1"
    - "tier:compute"
    - "role:execution"
  heartbeat_interval: "30s"
  registration_retry_interval: "60s"

api:
  base_url: "https://agents.stavily.com"
  timeout: "30s"
  retry:
    max_attempts: 3
    backoff: "exponential"
    initial_interval: "1s"
    max_interval: "60s"
  auth:
    type: "certificate"
    cert_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/client.crt"
    key_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/client.key"
    ca_file: "/opt/stavily/agent-{AGENT_ID}/config/certificates/ca.crt"
  rate_limit:
    requests_per_second: 10
    burst: 20

logging:
  level: "info"
  format: "json"
  file: "/opt/stavily/agent-{AGENT_ID}/logs/action-agent.log"
  max_size: 100
  max_backups: 5
  max_age: 30
  compress: true

security:
  sandbox:
    enabled: true
    user: "stavily"
    chroot: "/opt/stavily"
  tls:
    enabled: true
    min_version: "1.3"
    cipher_suites: []
  audit:
    enabled: true
    file: "/opt/stavily/agent-{AGENT_ID}/logs/audit/action-audit.log"

plugins:
  dir: "/opt/stavily/agent-{AGENT_ID}/data/plugins"
  config_dir: "/opt/stavily/agent-{AGENT_ID}/config/plugins"
  auto_update: true
  update_interval: "1h"
  max_memory: "512MB"
  timeout: "10m"
  allowed_plugins:
    - "service-restart"
    - "shell-command"
    - "python-script"
    - "file-operations"
    - "api-call"
  blocked_plugins: []

health:
  enabled: true
  port: 8081
  path: "/health"
  bind: "0.0.0.0"

metrics:
  enabled: true
  port: 9091
  path: "/metrics"
  bind: "127.0.0.1"
  namespace: "stavily_action"

# Action-specific settings
action:
  poll_interval: "5s"
  max_concurrent_actions: 3
  action_timeout: "30m"
  result_buffer_size: 100
  work_dir: "/opt/stavily/agent-{AGENT_ID}/tmp/workdir"
  max_output_size: "10MB"
  cleanup_interval: "1h"
```

## Python Plugin Architecture

**IMPORTANT**: Stavily agents are designed to execute Python plugins, not Go plugins. Each compiled agent (sensor or action) can execute any compatible Python plugin.

### Plugin Execution Model

- **Universal Agents**: One compiled agent binary can execute multiple Python plugins
- **Runtime**: Agents include Python runtime for plugin execution
- **Communication**: Plugins communicate with agents via JSON over stdin/stdout
- **Isolation**: Each plugin runs in a sandboxed environment with resource limits

### Plugin Types

- **Trigger Plugins** (Sensor Agents): Monitor conditions and generate events
- **Action Plugins** (Action Agents): Execute operations and return results

See `/examples/plugins/` for Python plugin examples and `/examples/configs/` for standardized configuration templates.

## Environment Variables

Environment variables should be used **only for sensitive data** like authentication tokens. All other configuration should be in YAML files.

### Required Environment Variables

```bash
# Authentication token (required)
export STAVILY_AGENT_TOKEN="your-jwt-token-here"
```

### Optional Environment Variables

```bash
# Demo mode for testing plugins (default: true)
export STAVILY_DEMO_MODE="true"

# Override organization ID
export STAVILY_ORGANIZATION_ID="your-org-id"

# Override API base URL
export STAVILY_API_BASE_URL="https://your-api.com"
```

### Legacy Environment Variables (Deprecated)

The following environment variables are no longer recommended and should be moved to configuration files:

### Agent Settings

```bash
# Basic agent settings
export STAVILY_AGENT_ID="sensor-001"
export STAVILY_AGENT_NAME="My Sensor Agent"
export STAVILY_AGENT_TYPE="sensor"
export STAVILY_AGENT_ORGANIZATION_ID="org-123"
export STAVILY_AGENT_BASE_DIR="/opt/stavily"
export STAVILY_AGENT_ENVIRONMENT="production"
export STAVILY_AGENT_HEARTBEAT_INTERVAL="30s"
export STAVILY_AGENT_REGISTRATION_RETRY_INTERVAL="60s"

# Agent tags (comma-separated)
export STAVILY_AGENT_TAGS="datacenter:us-east-1,tier:web,role:monitoring"
```

### API Settings

```bash
# API configuration
export STAVILY_API_BASE_URL="https://agents.stavily.com"
export STAVILY_API_TIMEOUT="30s"

# Retry settings
export STAVILY_API_RETRY_MAX_ATTEMPTS="3"
export STAVILY_API_RETRY_BACKOFF="exponential"
export STAVILY_API_RETRY_INITIAL_INTERVAL="1s"
export STAVILY_API_RETRY_MAX_INTERVAL="60s"

# Authentication
export STAVILY_API_AUTH_TYPE="certificate"
export STAVILY_API_AUTH_CERT_FILE="/certs/client.crt"
export STAVILY_API_AUTH_KEY_FILE="/certs/client.key"
export STAVILY_API_AUTH_CA_FILE="/certs/ca.crt"

# JWT authentication (alternative)
export STAVILY_API_AUTH_TYPE="jwt"
export STAVILY_API_AUTH_TOKEN_FILE="/tokens/agent.jwt"
export STAVILY_API_AUTH_REFRESH_THRESHOLD="5m"

# Rate limiting
export STAVILY_API_RATE_LIMIT_REQUESTS_PER_SECOND="10"
export STAVILY_API_RATE_LIMIT_BURST="20"
```

### Logging Settings

```bash
# Logging configuration
export STAVILY_LOGGING_LEVEL="info"
export STAVILY_LOGGING_FORMAT="json"
export STAVILY_LOGGING_FILE="/var/log/stavily/agent.log"
export STAVILY_LOGGING_MAX_SIZE="100"
export STAVILY_LOGGING_MAX_BACKUPS="5"
export STAVILY_LOGGING_MAX_AGE="30"
export STAVILY_LOGGING_COMPRESS="true"
```

### Security Settings

```bash
# Sandbox settings
export STAVILY_SECURITY_SANDBOX_ENABLED="true"
export STAVILY_SECURITY_SANDBOX_USER="stavily"
export STAVILY_SECURITY_SANDBOX_CHROOT="/opt/stavily"

# TLS settings
export STAVILY_SECURITY_TLS_ENABLED="true"
export STAVILY_SECURITY_TLS_MIN_VERSION="1.3"

# Audit settings
export STAVILY_SECURITY_AUDIT_ENABLED="true"
export STAVILY_SECURITY_AUDIT_FILE="/var/log/stavily/audit.log"
```

### Plugin Settings

```bash
# Plugin configuration
export STAVILY_PLUGINS_DIR="/opt/stavily/plugins"
export STAVILY_PLUGINS_CONFIG_DIR="/opt/stavily/config/plugins"
export STAVILY_PLUGINS_AUTO_UPDATE="true"
export STAVILY_PLUGINS_UPDATE_INTERVAL="1h"
export STAVILY_PLUGINS_MAX_MEMORY="256MB"
export STAVILY_PLUGINS_TIMEOUT="5m"

# Plugin allowlist/blocklist (comma-separated)
export STAVILY_PLUGINS_ALLOWED_PLUGINS="prometheus-trigger,file-watcher-trigger"
export STAVILY_PLUGINS_BLOCKED_PLUGINS="debug-plugin,test-plugin"
```

### Health and Metrics Settings

```bash
# Health check settings
export STAVILY_HEALTH_ENABLED="true"
export STAVILY_HEALTH_PORT="8080"
export STAVILY_HEALTH_PATH="/health"
export STAVILY_HEALTH_BIND="0.0.0.0"

# Metrics settings
export STAVILY_METRICS_ENABLED="true"
export STAVILY_METRICS_PORT="9090"
export STAVILY_METRICS_PATH="/metrics"
export STAVILY_METRICS_BIND="127.0.0.1"
export STAVILY_METRICS_NAMESPACE="stavily_agent"
```

### Agent-Specific Settings

```bash
# Sensor agent specific
export STAVILY_SENSOR_TRIGGER_CHECK_INTERVAL="10s"
export STAVILY_SENSOR_MAX_CONCURRENT_TRIGGERS="5"
export STAVILY_SENSOR_TRIGGER_TIMEOUT="30s"
export STAVILY_SENSOR_EVENT_BUFFER_SIZE="1000"
export STAVILY_SENSOR_BATCH_SIZE="10"
export STAVILY_SENSOR_FLUSH_INTERVAL="5s"

# Action agent specific
export STAVILY_ACTION_POLL_INTERVAL="5s"
export STAVILY_ACTION_MAX_CONCURRENT_ACTIONS="3"
export STAVILY_ACTION_ACTION_TIMEOUT="30m"
export STAVILY_ACTION_RESULT_BUFFER_SIZE="100"
export STAVILY_ACTION_WORK_DIR="/tmp/stavily-work"
export STAVILY_ACTION_MAX_OUTPUT_SIZE="10MB"
export STAVILY_ACTION_CLEANUP_INTERVAL="1h"
```

## Advanced Configuration Scenarios

### 1. Multi-Environment Configuration

#### Development Environment

```yaml
# dev-config.yaml
agent:
  id: "sensor-dev-${HOSTNAME}"
  name: "Development Sensor - ${HOSTNAME}"
  type: "sensor"
  organization_id: "org-dev"
  environment: "development"
  heartbeat_interval: "10s"

api:
  base_url: "https://dev-agents.stavily.com"
  timeout: "10s"
  auth:
    type: "jwt"
    token_file: "/dev/secrets/dev-token"

logging:
  level: "debug"
  format: "text"
  file: "/tmp/stavily-dev.log"

security:
  sandbox:
    enabled: false
  tls:
    enabled: false
  audit:
    enabled: false

plugins:
  auto_update: false
  timeout: "1m"
  allowed_plugins: ["*"]
```

#### Staging Environment

```yaml
# staging-config.yaml
agent:
  id: "sensor-staging-${HOSTNAME}"
  name: "Staging Sensor - ${HOSTNAME}"
  type: "sensor"
  organization_id: "org-staging"
  environment: "staging"
  heartbeat_interval: "20s"

api:
  base_url: "https://staging-agents.stavily.com"
  timeout: "20s"
  auth:
    type: "certificate"
    cert_file: "/etc/stavily/staging-client.crt"
    key_file: "/etc/stavily/staging-client.key"
    ca_file: "/etc/stavily/staging-ca.crt"

logging:
  level: "info"
  format: "json"
  file: "/var/log/stavily/staging.log"

security:
  sandbox:
    enabled: true
  tls:
    enabled: true
    min_version: "1.2"
  audit:
    enabled: true
    file: "/var/log/stavily/staging-audit.log"

plugins:
  auto_update: true
  update_interval: "2h"
  timeout: "3m"
```

#### Production Environment

```yaml
# prod-config.yaml
agent:
  id: "sensor-prod-${DATACENTER}-${HOSTNAME}"
  name: "Production Sensor - ${DATACENTER} - ${HOSTNAME}"
  type: "sensor"
  organization_id: "org-prod"
  environment: "production"
  tags:
    - "datacenter:${DATACENTER}"
    - "environment:production"
    - "criticality:high"
  heartbeat_interval: "30s"

api:
  base_url: "https://agents.stavily.com"
  timeout: "30s"
  retry:
    max_attempts: 5
    backoff: "exponential"
    initial_interval: "2s"
    max_interval: "120s"
  auth:
    type: "certificate"
    cert_file: "/opt/stavily/certs/prod-client.crt"
    key_file: "/opt/stavily/certs/prod-client.key"
    ca_file: "/opt/stavily/certs/prod-ca.crt"
  rate_limit:
    requests_per_second: 20
    burst: 50

logging:
  level: "warn"
  format: "json"
  file: "/var/log/stavily/production.log"
  max_size: 500
  max_backups: 10
  max_age: 90
  compress: true

security:
  sandbox:
    enabled: true
    user: "stavily"
    chroot: "/opt/stavily"
  tls:
    enabled: true
    min_version: "1.3"
    cipher_suites:
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
  audit:
    enabled: true
    file: "/var/log/stavily/production-audit.log"

plugins:
  auto_update: false  # Manual updates in production
  timeout: "10m"
  max_memory: "1GB"
  allowed_plugins:
    - "prometheus-trigger"
    - "file-watcher-trigger"
    - "shell-action"
    - "email-output"
```

### 2. High-Security Configuration

```yaml
# high-security-config.yaml
agent:
  id: "sensor-secure-${HOSTNAME}"
  name: "High Security Sensor"
  type: "sensor"
  organization_id: "org-secure"
  environment: "secure"

api:
  base_url: "https://secure-agents.stavily.com"
  timeout: "60s"
  retry:
    max_attempts: 10
    backoff: "exponential"
    initial_interval: "5s"
    max_interval: "300s"
  auth:
    type: "certificate"
    cert_file: "/etc/pki/stavily/client.crt"
    key_file: "/etc/pki/stavily/client.key"
    ca_file: "/etc/pki/stavily/ca.crt"

logging:
  level: "info"
  format: "json"
  file: "/var/log/stavily/secure.log"
  max_size: 100
  max_backups: 20
  max_age: 365
  compress: true

security:
  sandbox:
    enabled: true
    user: "stavily-secure"
    chroot: "/opt/stavily-secure"
  tls:
    enabled: true
    min_version: "1.3"
    cipher_suites:
      - "TLS_AES_256_GCM_SHA384"
  audit:
    enabled: true
    file: "/var/log/stavily/secure-audit.log"

plugins:
  dir: "/opt/stavily-secure/plugins"
  config_dir: "/opt/stavily-secure/config/plugins"
  auto_update: false
  timeout: "2m"
  max_memory: "128MB"
  allowed_plugins:
    - "prometheus-trigger"
    - "file-watcher-trigger"
  blocked_plugins:
    - "shell-action"
    - "python-script-action"

health:
  enabled: true
  port: 8080
  bind: "127.0.0.1"  # Only localhost access

metrics:
  enabled: true
  port: 9090
  bind: "127.0.0.1"  # Only localhost access
```

### 3. Container-Optimized Configuration

```yaml
# container-config.yaml
agent:
  id: "${CONTAINER_ID}"
  name: "Container Agent - ${CONTAINER_NAME}"
  type: "${AGENT_TYPE}"
  organization_id: "${ORG_ID}"
  base_dir: "/app/agent-{AGENT_ID}"

api:
  base_url: "${STAVILY_API_URL}"
  timeout: "15s"
  auth:
    type: "jwt"
    token_file: "/run/secrets/stavily-token"

logging:
  level: "${LOG_LEVEL:-info}"
  format: "json"
  # Log to stdout for container log collection
  file: "/dev/stdout"

security:
  sandbox:
    enabled: false  # Container provides isolation
  tls:
    enabled: true
    min_version: "1.3"
  audit:
    enabled: true
    file: "/dev/stdout"

plugins:
  dir: "/app/agent-{AGENT_ID}/plugins"
  config_dir: "/app/agent-{AGENT_ID}/config/plugins"
  auto_update: true
  timeout: "5m"
  max_memory: "${PLUGIN_MAX_MEMORY:-256MB}"

health:
  enabled: true
  port: 8080
  bind: "0.0.0.0"

metrics:
  enabled: true
  port: 9090
  bind: "0.0.0.0"
```

### 4. Plugin-Specific Configuration

#### Prometheus Trigger Plugin

```yaml
# config/plugins/prometheus-trigger.yaml
plugin:
  name: "prometheus-trigger"
  version: "1.2.0"
  enabled: true

prometheus:
  url: "http://prometheus:9090"
  timeout: "30s"
  auth:
    type: "basic"
    username: "stavily"
    password_file: "/secrets/prometheus-password"

queries:
  - name: "high-cpu"
    query: "cpu_usage_percent > 80"
    interval: "30s"
    threshold: 3
    labels:
      severity: "warning"
      component: "cpu"
  
  - name: "disk-full"
    query: "disk_usage_percent > 90"
    interval: "60s"
    threshold: 1
    labels:
      severity: "critical"
      component: "disk"

  - name: "memory-pressure"
    query: "memory_usage_percent > 85"
    interval: "30s"
    threshold: 2
    labels:
      severity: "warning"
      component: "memory"
```

#### Email Output Plugin

```yaml
# config/plugins/email-output.yaml
plugin:
  name: "email-output"
  version: "1.1.0"
  enabled: true

smtp:
  host: "smtp.company.com"
  port: 587
  encryption: "starttls"
  auth:
    username: "alerts@company.com"
    password_file: "/secrets/smtp-password"

templates:
  default:
    subject: "[Stavily Alert] {{ .Severity }}: {{ .Title }}"
    body: |
      Alert: {{ .Title }}
      Severity: {{ .Severity }}
      Time: {{ .Timestamp }}
      
      Description: {{ .Description }}
      
      Agent: {{ .Agent.Name }} ({{ .Agent.ID }})
      Environment: {{ .Agent.Environment }}
      
      Details:
      {{ range $key, $value := .Labels }}
      - {{ $key }}: {{ $value }}
      {{ end }}

recipients:
  default:
    - "ops-team@company.com"
  critical:
    - "ops-team@company.com"
    - "on-call@company.com"
    - "cto@company.com"
  warning:
    - "ops-team@company.com"
```

## Configuration Validation

### Built-in Validation

The agents perform automatic validation on startup:

```bash
# Validate configuration without starting
sensor-agent --config=/path/to/config.yaml --validate

# Check configuration and show effective values
sensor-agent --config=/path/to/config.yaml --show-config
```

### Manual Validation Script

```bash
#!/bin/bash
# validate-config.sh

CONFIG_FILE="$1"
if [ -z "$CONFIG_FILE" ]; then
    echo "Usage: $0 <config-file>"
    exit 1
fi

echo "Validating Stavily agent configuration: $CONFIG_FILE"

# Check file exists and is readable
if [ ! -r "$CONFIG_FILE" ]; then
    echo "ERROR: Configuration file not found or not readable"
    exit 1
fi

# Validate YAML syntax
if ! yq eval '.' "$CONFIG_FILE" > /dev/null 2>&1; then
    echo "ERROR: Invalid YAML syntax"
    exit 1
fi

# Check required fields
REQUIRED_FIELDS=(
    ".agent.id"
    ".agent.name"
    ".agent.type"
    ".agent.organization_id"
    ".api.base_url"
    ".api.auth.type"
)

for field in "${REQUIRED_FIELDS[@]}"; do
    if ! yq eval "$field" "$CONFIG_FILE" > /dev/null 2>&1; then
        echo "ERROR: Required field missing: $field"
        exit 1
    fi
done

# Validate agent type
AGENT_TYPE=$(yq eval '.agent.type' "$CONFIG_FILE")
if [[ "$AGENT_TYPE" != "sensor" && "$AGENT_TYPE" != "action" ]]; then
    echo "ERROR: Invalid agent type: $AGENT_TYPE (must be 'sensor' or 'action')"
    exit 1
fi

# Validate auth configuration
AUTH_TYPE=$(yq eval '.api.auth.type' "$CONFIG_FILE")
case "$AUTH_TYPE" in
    "certificate")
        for cert_field in ".api.auth.cert_file" ".api.auth.key_file" ".api.auth.ca_file"; do
            if ! yq eval "$cert_field" "$CONFIG_FILE" > /dev/null 2>&1; then
                echo "ERROR: Certificate auth requires: $cert_field"
                exit 1
            fi
        done
        ;;
    "jwt")
        if ! yq eval '.api.auth.token_file' "$CONFIG_FILE" > /dev/null 2>&1; then
            echo "ERROR: JWT auth requires: .api.auth.token_file"
            exit 1
        fi
        ;;
    *)
        echo "ERROR: Invalid auth type: $AUTH_TYPE (must be 'certificate' or 'jwt')"
        exit 1
        ;;
esac

# Check certificate files exist (if using certificate auth)
if [ "$AUTH_TYPE" = "certificate" ]; then
    CERT_FILE=$(yq eval '.api.auth.cert_file' "$CONFIG_FILE")
    KEY_FILE=$(yq eval '.api.auth.key_file' "$CONFIG_FILE")
    CA_FILE=$(yq eval '.api.auth.ca_file' "$CONFIG_FILE")
    
    for file in "$CERT_FILE" "$KEY_FILE" "$CA_FILE"; do
        if [ ! -r "$file" ]; then
            echo "WARNING: Certificate file not found or not readable: $file"
        fi
    done
fi

# Validate base directory
BASE_DIR=$(yq eval '.agent.base_dir // "agent-{AGENT_ID}"' "$CONFIG_FILE")
if [ ! -d "$BASE_DIR" ]; then
    echo "WARNING: Base directory does not exist: $BASE_DIR"
fi

# Validate ports
HEALTH_PORT=$(yq eval '.health.port // 8080' "$CONFIG_FILE")
METRICS_PORT=$(yq eval '.metrics.port // 9090' "$CONFIG_FILE")

if [ "$HEALTH_PORT" = "$METRICS_PORT" ]; then
    echo "ERROR: Health and metrics ports cannot be the same"
    exit 1
fi

# Check port availability
if netstat -tuln | grep -q ":$HEALTH_PORT "; then
    echo "WARNING: Health port $HEALTH_PORT is already in use"
fi

if netstat -tuln | grep -q ":$METRICS_PORT "; then
    echo "WARNING: Metrics port $METRICS_PORT is already in use"
fi

echo "Configuration validation completed successfully"
```

### Configuration Testing

```bash
#!/bin/bash
# test-config.sh

CONFIG_FILE="$1"
AGENT_TYPE="$2"

# Start agent in test mode
timeout 30s ${AGENT_TYPE}-agent \
    --config="$CONFIG_FILE" \
    --test-mode \
    --log-level=debug

if [ $? -eq 0 ]; then
    echo "Configuration test passed"
else
    echo "Configuration test failed"
    exit 1
fi
```

This comprehensive configuration guide covers all aspects of configuring Stavily agents, from basic setup to advanced scenarios. Use the validation tools to ensure your configuration is correct before deployment. 