# Directory Structure and Naming Update

## Overview

This document summarizes the current Stavily agent directory structure, which is now automatically created at agent startup. All agents use Go 1.24.4 and the enhanced-agent has been fully removed.

## Key Changes

### 1. Base Directory Structure (Auto-Creation)
- **Now**: Directory tree is created automatically by the agent on startup using the configuration in `shared/pkg/config/config.go`.
- **Structure:**
```
{base_folder}/
├── config/
│   ├── plugins/
│   └── certificates/
├── data/
│   ├── plugins/
│   ├── cache/
│   └── state/
├── logs/
│   ├── plugins/
│   └── audit/
└── tmp/
    └── workdir/
```
- **Helper Methods:**
  - `GetDataDir()`, `GetStateDir()`, `GetTmpDir()`, `GetWorkDir()`

### 2. SystemD Service Names
- `sensor-agent-{AGENT_ID}.service`, `action-agent-{AGENT_ID}.service`

### 3. Migration Guide
- No manual directory creation needed; agents will create all required folders on first run.

### 4. Go Version
- All modules require **Go 1.24.4**

### 5. Enhanced-Agent Removal
- All references and configs for enhanced-agent have been removed as of the 2025 refactor.

## See also
- `docs/CONFIGURATION_GUIDE.md` for config details
- `shared/pkg/config/config.go` for implementation

## Files Updated

### Documentation
- `README.md` - Added new section explaining directory structure and naming
- `docs/CONFIGURATION_GUIDE.md` - Updated all path references
- `docs/DEPLOYMENT_GUIDE.md` - Updated systemd service names and paths
- `QUICKSTART.md` - Updated directory references
- `IMPLEMENTATION_SUMMARY.md` - Added multi-agent support note

### Configuration
- `shared/pkg/config/validation.go` - Updated test file naming
- All example configurations updated to use new paths

### Docker & Deployment
- `docker-compose.yml` - Simplified to core agents only (sensor + action)
- `deployments/docker/docker-compose.dev.yml` - Development environment
- Updated Kubernetes manifest examples

## Benefits

1. **Multiple Agents**: Run multiple sensor or action agents on the same machine
2. **Clear Separation**: Each agent has its own isolated directory and service
3. **Easy Management**: Systemd services clearly identify which agent they control
4. **Scalability**: Support for complex deployment scenarios
5. **Maintenance**: Easier to manage configurations, logs, and data per agent

## Configuration Template

```yaml
agent:
  id: "sensor-web-01"                      # Unique agent ID
  name: "Web Server Sensor"
  type: "sensor"
  organization_id: "org-123"
  base_dir: "/var/lib/stavily/agent-{AGENT_ID}"  # Agent-specific directory

# Rest of configuration remains the same
```

## Service Management

```bash
# Start specific agent
sudo systemctl start sensor-agent-web-01.service

# Check status
sudo systemctl status sensor-agent-web-01.service

# View logs
sudo journalctl -u sensor-agent-web-01.service -f

# Enable on boot
sudo systemctl enable sensor-agent-web-01.service
```

## Additional Cleanup

As part of this update, all monitoring and mocking infrastructure has been removed from the core docker-compose.yml to keep it focused on the essential agent services only:

- ✅ **Removed**: Prometheus monitoring service
- ✅ **Removed**: Grafana dashboard service  
- ✅ **Removed**: Mock orchestrator service
- ✅ **Removed**: All monitoring-related configurations and documentation references
- ✅ **Simplified**: Core docker-compose.yml to include only sensor-agent and action-agent services

The development environment (`deployments/docker/docker-compose.dev.yml`) can still be used for extended development features if needed.

This update provides the foundation for more flexible and scalable agent deployments while maintaining backward compatibility through configuration.
