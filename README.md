# Stavily Agents

Light-weight **Sensor** and **Action** agents written in Go. They run on customer infrastructure, poll the Stavily Orchestrator, and execute sandboxed Python plugins.

> **Latest update – Jul 2025:** Full plugin life-cycle: `plugin_install`, **`plugin_update`**, `execute`, plus `plugin_version` (branch / tag / commit) selection. See the refactor summary for full details.

---
## Directory Layout
```text
01-Agents/
├── shared/            # Common libraries (agent core, API client, plugin system)
├── sensor-agent/      # Sensor agent implementation
├── action-agent/      # Action agent implementation
├── bin/               # Built binaries (created by `make build`)
└── configs/           # Sample YAML configs
```

---
## Quick Start
```bash
# Build both agents
make build

# Run Sensor agent with example config
./bin/sensor-agent  --config configs/dev-sensor.yaml

# Run Action agent with example config
./bin/action-agent  --config configs/dev-action.yaml
```
Check health:
```bash
curl http://localhost:8080/health   # Sensor
curl http://localhost:8081/health   # Action
```

---
## Recent Updates (2025)
- **Plugin life-cycle** – install, **update**, execute.
- **`plugin_version`** – choose branch / tag / commit.
- **Factory pattern** – single source for downloaders & executors (DRY).
- **Unified EnhancedPluginManager** used by both agents.
- Rich validation, detailed processing logs, auto-cleanup on failure.

---
## Security Highlights
- mTLS communication
- Non-root execution & filesystem sandbox
- Strict tenant isolation

---
## Docs & Support
- Docs: <https://docs.stavily.com/agents>
- Community: <https://community.stavily.com>
- Issues: <https://github.com/Stavily/01-Agents/issues> 

---
## Deployment ‑ Quick Guide
### 1. Bare-metal (system service)
```bash
# Install binary (example for sensor)
curl -L https://github.com/Stavily/01-Agents/releases/latest/download/sensor-agent-linux-amd64 \
     -o /usr/local/bin/sensor-agent && chmod +x /usr/local/bin/sensor-agent

# Create minimal config
a mkdir -p /etc/stavily && tee /etc/stavily/agent.yaml << 'EOF'
agent:
  id: "sensor-001"
  type: "sensor"
  base_dir: "/var/lib/stavily"
api:
  base_url: "https://agents.stavily.com"
  auth:
    type: "certificate"
    cert_file: "/etc/stavily/certs/client.crt"
    key_file:  "/etc/stavily/certs/client.key"
    ca_file:   "/etc/stavily/certs/ca.crt"
logging:
  level: "info"
EOF

# Run (foreground)
sensor-agent --config /etc/stavily/agent.yaml
```

### 2. Docker (recommended)
```bash
# Host directories
export AGENT_HOME=$HOME/stavily-agent
mkdir -p $AGENT_HOME/{config,certs,data,logs}

# Drop sample config
cp configs/dev-sensor.yaml $AGENT_HOME/config/agent.yaml

# Run container
docker run -d --name stavily-sensor \
  -v $AGENT_HOME:/app/agent:rw \
  -p 8080:8080 \
  stavily/sensor-agent:latest \
  --config=/app/agent/config/agent.yaml
```

The same steps apply to the **Action Agent** – replace the image/binary names and ports.

--- 