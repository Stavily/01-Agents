# Agent Configuration Templates

This directory contains configuration templates for connecting Stavily agents to the orchestrator.

## Templates

### `action-agent-localhost.yaml`
Configuration template for Action Agents connecting to localhost:8000 orchestrator.

**Features:**
- Connects to `http://localhost:8000`
- 512MB memory limit for action plugins
- 30-minute timeout for action execution
- Network access enabled for actions
- Health check on port 8081
- Metrics on port 9091

### `sensor-agent-localhost.yaml`
Configuration template for Sensor Agents connecting to localhost:8000 orchestrator.

**Features:**
- Connects to `http://localhost:8000`
- 256MB memory limit for sensor plugins
- 5-minute timeout for sensor operations
- Network access disabled (security)
- Health check on port 8080
- Metrics on port 9090

## Quick Setup

Use the automated setup script:

```bash
# Setup both agents for localhost:8000
sudo ./scripts/setup-localhost-agents.sh

# Test orchestrator connectivity
./scripts/setup-localhost-agents.sh test

# Clean up (remove all agent directories)
sudo ./scripts/setup-localhost-agents.sh clean
```

## Manual Setup

### 1. Environment Variables
```bash
export STAVILY_ORG_ID="your-org-id"
export STAVILY_ENV="production"  # or "dev"
```

### 2. Create Agent Directory Structure
```bash
# For Action Agent
sudo mkdir -p /opt/stavily/agent-action-$(hostname)-001/{config/{certificates,plugins},data/plugins,logs/audit,tmp/workdir,cache/plugins}

# For Sensor Agent  
sudo mkdir -p /opt/stavily/agent-sensor-$(hostname)-001/{config/{certificates,plugins},data/plugins,logs/audit,tmp/workdir,cache/plugins}
```

### 3. Generate Configuration
```bash
# Replace template variables and copy to agent directory
sed -e "s/\${HOSTNAME}/$(hostname)/g" \
    -e "s/your-org-id/${STAVILY_ORG_ID}/g" \
    configs/action-agent-localhost.yaml > /opt/stavily/agent-action-$(hostname)-001/config/agent.yaml
```

### 4. Create JWT Token
```bash
# Development token (replace with proper JWT in production)
echo "dev.eyJzdWIiOiJhY3Rpb24taG9zdC0wMDEiLCJpc3MiOiJzdGF2aWx5LW9yY2hlc3RyYXRvciJ9.dev" > \
    /opt/stavily/agent-action-$(hostname)-001/config/certificates/agent.jwt
```

## Directory Structure

After setup, each agent will have:

```
/opt/stavily/agent-{TYPE}-{HOSTNAME}-001/
├── config/
│   ├── agent.yaml           # Main configuration
│   ├── certificates/        # JWT tokens and TLS certificates
│   │   ├── agent.jwt
│   │   ├── agent.crt
│   │   ├── agent.key
│   │   └── ca.crt
│   └── plugins/            # Plugin configurations
├── data/
│   └── plugins/            # Plugin binaries
├── logs/
│   ├── agent.log          # Main agent log
│   └── audit/             # Audit logs
├── tmp/
│   └── workdir/          # Action execution workspace
└── cache/
    └── plugins/          # Plugin cache
```

## Security Configuration

### TLS Settings
- **Enabled**: Yes (with self-signed certs for localhost)
- **Min Version**: TLS 1.3
- **Server Name**: localhost
- **Skip Verify**: true (for development only)

### Authentication
- **Method**: JWT Bearer token
- **Token Location**: `config/certificates/agent.jwt`
- **TTL**: 1 hour

### Sandbox Settings
- **Action Agent**: 512MB RAM, 1 CPU core, network enabled
- **Sensor Agent**: 256MB RAM, 0.5 CPU core, network disabled
- **Execution Timeout**: 30m (action), 5m (sensor)

## Validation

### Health Checks
```bash
# Sensor Agent
curl http://localhost:8080/health

# Action Agent  
curl http://localhost:8081/health
```

### Metrics
```bash
# Sensor Agent
curl http://localhost:9090/metrics

# Action Agent
curl http://localhost:9091/metrics
```

### Agent Registration
```bash
# Check if agents are registered with orchestrator
curl -H "Authorization: Bearer $JWT_TOKEN" \
     http://localhost:8000/api/v1/agents/status
```

## Troubleshooting

### Common Issues

1. **Permission Denied**
   ```bash
   sudo chown -R stavily:stavily /opt/stavily/
   sudo chmod 750 /opt/stavily/agent-*/
   ```

2. **Connection Refused**
   - Ensure orchestrator is running: `cd 02-Orchestrator && python main.py`
   - Check firewall settings for localhost:8000

3. **JWT Token Issues**
   - Verify token file exists and is readable
   - Check token expiration in development setup
   - Ensure proper permissions (600) on JWT file

4. **Plugin Loading Errors**
   - Check plugin directory permissions
   - Verify plugin allowlist in configuration
   - Check sandbox resource limits

### Log Locations
- **Agent Logs**: `/opt/stavily/agent-*/logs/agent.log`
- **Audit Logs**: `/opt/stavily/agent-*/logs/audit/`
- **System Logs**: `sudo journalctl -u stavily-*-agent`

## Next Steps

1. **Start Orchestrator**: `cd 02-Orchestrator && python main.py`
2. **Build Agents**: `cd 01-Agents && make build`
3. **Install Agents**: Copy binaries to `/usr/local/bin/`
4. **Start Services**: `sudo systemctl start stavily-*-agent`
5. **Verify Connection**: Check agent registration in orchestrator logs 