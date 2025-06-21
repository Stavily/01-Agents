# Directory Structure and Naming Update

## Overview

This document summarizes the changes made to support multiple agents of the same type on a single machine by implementing agent-specific directory structures and systemd service naming.

## Key Changes

### 1. Base Directory Structure
- **Before**: `.stavily` (single directory for all agents)
- **After**: `agent-{AGENT_ID}` (unique directory per agent)

### 2. SystemD Service Names
- **Before**: `stavily-sensor.service`, `stavily-action.service`
- **After**: `sensor-agent-{AGENT_ID}.service`, `action-agent-{AGENT_ID}.service`

### 3. Directory Examples

#### Old Structure (Single Agent)
```
/var/lib/stavily/
└── .stavily/
    ├── config/
    ├── data/
    └── logs/
```

#### New Structure (Multiple Agents)
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

### 4. SystemD Service Examples
- `/etc/systemd/system/sensor-agent-web-01.service`
- `/etc/systemd/system/sensor-agent-db-01.service`
- `/etc/systemd/system/action-agent-exec-01.service`

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

## Migration Guide

### For Existing Deployments
1. Stop existing agents
2. Move `.stavily` directory to `agent-{AGENT_ID}`
3. Update systemd service files with new names and paths
4. Update configuration files
5. Restart agents with new service names

### For New Deployments
Simply follow the updated documentation - all examples now use the new structure.

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
