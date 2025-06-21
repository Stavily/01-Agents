# Stavily Agents Examples

This directory contains example configurations and plugins for Stavily agents.

## Directory Structure

```
examples/
├── configs/              # Standard configuration examples
│   ├── sensor-agent.yaml    # Sensor agent configuration template
│   └── action-agent.yaml    # Action agent configuration template
└── plugins/              # Python plugin examples
    ├── cpu-monitor/          # CPU monitoring trigger plugin
    │   ├── cpu_monitor.py
    │   ├── plugin.yaml
    │   └── requirements.txt
    └── service-restart/      # Service restart action plugin
        ├── service_restart.py
        ├── plugin.yaml
        └── requirements.txt
```

## Configuration Examples

### Sensor Agent Configuration

The `configs/sensor-agent.yaml` file provides a standardized configuration template for sensor agents. Key features:

- **Standardized structure** following CONFIGURATION_GUIDE.md
- **JWT authentication** with token file
- **Python plugin support** with resource limits
- **Comprehensive logging** with rotation
- **Health checks and metrics** endpoints

### Action Agent Configuration

The `configs/action-agent.yaml` file provides a standardized configuration template for action agents. Key features:

- **Action execution framework** with timeouts and concurrency limits
- **Python plugin support** for action plugins
- **Security sandbox** configuration
- **Work directory** for temporary files
- **Result buffering** and cleanup

## Plugin Examples

### CPU Monitor Plugin (Trigger)

**Location**: `plugins/cpu-monitor/`

A Python trigger plugin that monitors CPU usage and generates alerts when thresholds are exceeded.

**Features**:
- Configurable CPU usage threshold (default: 80%)
- Adjustable monitoring interval (default: 30s)
- Detailed system information in events
- Resource-efficient monitoring using `psutil`

**Configuration**:
```yaml
threshold: 85.0    # CPU usage percentage threshold
interval: 30       # Check interval in seconds
```

**Usage**:
```bash
# Install dependencies
pip install -r plugins/cpu-monitor/requirements.txt

# Test the plugin
echo '{"action": "get_info"}' | python plugins/cpu-monitor/cpu_monitor.py
```

### Service Restart Plugin (Action)

**Location**: `plugins/service-restart/`

A Python action plugin that restarts system services using various service managers.

**Features**:
- Support for multiple service managers (systemctl, service, docker)
- Service allowlist for security
- Simulation mode for testing
- Comprehensive error handling and logging

**Configuration**:
```yaml
allowed_services:
  - "nginx"
  - "apache2"
  - "mysql"
service_manager: "systemctl"
```

**Usage**:
```bash
# Test the plugin
echo '{"action": "get_info"}' | python plugins/service-restart/service_restart.py

# Execute a restart action
echo '{
  "action": "execute_action",
  "action_request": {
    "id": "restart-001",
    "parameters": {
      "service_name": "nginx",
      "method": "systemctl"
    }
  }
}' | python plugins/service-restart/service_restart.py
```

## Plugin Architecture

### Python Plugin Interface

All Python plugins communicate with agents through JSON messages over stdin/stdout:

**Command Format**:
```json
{
  "action": "action_name",
  "config": { ... },
  "action_request": { ... }
}
```

**Response Format**:
```json
{
  "action": "action_name",
  "success": true|false,
  "data": { ... },
  "error": "error_message"
}
```

### Supported Actions

**Common Actions** (all plugins):
- `get_info` - Get plugin metadata
- `initialize` - Initialize with configuration
- `start` - Start plugin execution
- `stop` - Stop plugin execution
- `get_status` - Get current status
- `get_health` - Get health information

**Trigger Plugin Actions**:
- `detect_triggers` - Detect and return trigger events
- `get_trigger_config` - Get trigger configuration schema

**Action Plugin Actions**:
- `execute_action` - Execute an action with parameters
- `get_action_config` - Get action configuration schema

### Plugin Configuration File

Each plugin includes a `plugin.yaml` file with metadata and configuration:

```yaml
plugin:
  id: "plugin-id"
  name: "Plugin Name"
  description: "Plugin description"
  version: "1.0.0"
  type: "trigger|action"
  
  runtime:
    type: "python"
    version: "3.8+"
    entry_point: "main.py"
    requirements: "requirements.txt"
  
  configuration:
    # Plugin-specific config schema
  
  limits:
    memory: "128MB"
    cpu: "0.2"
    execution_time: "300s"
  
  permissions:
    network: false
    filesystem:
      read: []
      write: []
```

## Environment Variables

When using these configurations, set the following environment variables:

```bash
# Required: Agent authentication token
export STAVILY_AGENT_TOKEN="your-jwt-token-here"

# Optional: Enable demo mode for plugins (default: true)
export STAVILY_DEMO_MODE="true"
```

## Quick Start

1. **Copy configuration files**:
   ```bash
   cp examples/configs/sensor-agent.yaml sensor-agent/configs/production.yaml
   cp examples/configs/action-agent.yaml action-agent/configs/production.yaml
   ```

2. **Update configuration**:
   - Set your `organization_id`
   - Configure API endpoints
   - Adjust resource limits
   - Set appropriate file paths

3. **Install plugin dependencies**:
   ```bash
   pip install -r examples/plugins/cpu-monitor/requirements.txt
   ```

4. **Start agents**:
   ```bash
   export STAVILY_AGENT_TOKEN="your-token"
   docker-compose up -d
   ```

## Development

### Creating New Plugins

1. Create a new directory under `plugins/`
2. Implement the plugin following the interface pattern
3. Create `plugin.yaml` with metadata and configuration
4. Add `requirements.txt` for dependencies
5. Test using the JSON command interface

### Testing Plugins

Use the command-line interface to test plugins:

```bash
# Test plugin info
echo '{"action": "get_info"}' | python your_plugin.py

# Test initialization
echo '{"action": "initialize", "config": {"key": "value"}}' | python your_plugin.py

# Test plugin start
echo '{"action": "start"}' | python your_plugin.py
```

## Security Considerations

- **Resource Limits**: All plugins run with memory, CPU, and execution time limits
- **Filesystem Access**: Plugins have restricted filesystem access
- **Network Access**: Network access is disabled by default for most plugins
- **Service Allowlists**: Action plugins should use allowlists for security-critical operations
- **Sandboxing**: Agents run plugins in sandboxed environments

## Support

For more information:
- See `docs/CONFIGURATION_GUIDE.md` for detailed configuration options
- Check agent logs for plugin execution details
- Use health check endpoints to monitor plugin status 