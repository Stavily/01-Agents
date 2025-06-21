# Stavily Agents Refactoring Summary

This document summarizes the major refactoring changes made to the Stavily agents architecture to support Python plugins and standardized configuration.

## 🔄 **Major Changes Overview**

### 1. Plugin Architecture Transformation

**From**: Go-based compiled plugins
**To**: Python-based script plugins

- ❌ **Removed**: Go plugin examples (`cpu-monitor`, `service-restart`)
- ✅ **Added**: Python plugin framework with JSON communication
- ✅ **Added**: Plugin configuration files (`plugin.yaml`)
- ✅ **Added**: Resource limits and sandboxing for Python plugins

### 2. Configuration Standardization

**From**: Inconsistent configuration formats
**To**: Unified configuration following CONFIGURATION_GUIDE.md

- ✅ **Standardized**: Agent configuration structure
- ✅ **Updated**: Sensor agent config (`sensor-agent/configs/dev.yaml`)
- ✅ **Updated**: Action agent config (`action-agent/configs/dev.yaml`)
- ✅ **Created**: Standard configuration templates in `/examples/configs/`

### 3. Environment Variables Cleanup

**From**: Many configuration options in environment variables
**To**: Only sensitive data in environment variables

- ❌ **Removed**: `STAVILY_SENSOR_LOGGING_LEVEL`, `STAVILY_ACTION_LOGGING_LEVEL`, etc.
- ✅ **Kept**: `STAVILY_AGENT_TOKEN` for sensitive authentication
- ✅ **Added**: `STAVILY_DEMO_MODE` for plugin testing

### 4. Docker Compose Simplification

**From**: Complex setup with development tools
**To**: Clean agent-only setup

- ❌ **Removed**: Redis, MailHog development tools
- ❌ **Removed**: Unnecessary environment variables
- ✅ **Simplified**: Volume mappings
- ✅ **Added**: Plugin examples mounting

## 📁 **New File Structure**

```
01-Agents/
├── examples/
│   ├── configs/
│   │   ├── sensor-agent.yaml      # ✅ NEW: Standard sensor config
│   │   └── action-agent.yaml      # ✅ NEW: Standard action config
│   ├── plugins/
│   │   ├── cpu-monitor/
│   │   │   ├── cpu_monitor.py     # ✅ NEW: Python trigger plugin
│   │   │   ├── plugin.yaml        # ✅ NEW: Plugin metadata
│   │   │   └── requirements.txt   # ✅ NEW: Python dependencies
│   │   └── service-restart/
│   │       ├── service_restart.py # ✅ NEW: Python action plugin
│   │       ├── plugin.yaml        # ✅ NEW: Plugin metadata
│   │       └── requirements.txt   # ✅ NEW: Python dependencies
│   ├── env.example                # ✅ NEW: Environment variables template
│   └── README.md                  # ✅ NEW: Comprehensive examples guide
├── sensor-agent/configs/dev.yaml  # ✅ UPDATED: Standardized format
├── action-agent/configs/dev.yaml  # ✅ UPDATED: Standardized format
├── docker-compose.yml             # ✅ UPDATED: Simplified
├── deployments/docker/
│   └── docker-compose.dev.yml     # ✅ UPDATED: Removed dev tools
├── docs/
│   └── CONFIGURATION_GUIDE.md     # ✅ UPDATED: Python plugin info
└── REFACTORING_SUMMARY.md         # ✅ NEW: This document
```

## 🔧 **Configuration Changes**

### Sensor Agent Configuration

**Key Changes**:
- `tenant_id` → `organization_id`
- `heartbeat` → `heartbeat_interval`
- Added `registration_retry_interval`
- Simplified API configuration with `auth.type: "jwt"`
- Standardized plugin configuration
- Added sensor-specific settings section

### Action Agent Configuration

**Key Changes**:
- Added `organization_id` and `base_dir`
- Simplified API retry configuration
- Standardized plugin configuration
- Added action-specific settings section
- Removed complex development settings

### Environment Variables

**Before**:
```bash
STAVILY_SENSOR_LOGGING_LEVEL=debug
STAVILY_SENSOR_DEVELOPMENT_DEBUG_MODE=true
STAVILY_ACTION_LOGGING_LEVEL=info
STAVILY_ACTION_POLL_INTERVAL=30s
# ... many more
```

**After**:
```bash
STAVILY_AGENT_TOKEN=your-jwt-token-here
STAVILY_DEMO_MODE=true  # optional
```

## 🐍 **Python Plugin Architecture**

### Communication Protocol

Plugins communicate with agents via JSON over stdin/stdout:

**Command**:
```json
{
  "action": "detect_triggers",
  "config": {"threshold": 80.0}
}
```

**Response**:
```json
{
  "action": "detect_triggers",
  "success": true,
  "data": {
    "id": "cpu-high-123",
    "type": "cpu.high",
    "data": {"cpu_usage": 85.2}
  }
}
```

### Plugin Types

1. **Trigger Plugins** (Sensor Agents):
   - Monitor system conditions
   - Generate events when thresholds are met
   - Example: CPU monitor, disk space monitor

2. **Action Plugins** (Action Agents):
   - Execute system operations
   - Return execution results
   - Example: Service restart, file operations

### Resource Management

- Memory limits (e.g., 256MB for triggers, 512MB for actions)
- CPU limits with percentage allocation
- Execution timeouts
- Filesystem access restrictions
- Network access controls

## 🚀 **Benefits**

### 1. **Universal Agent Architecture**
- One compiled agent can execute any compatible Python plugin
- No need to recompile agents for new functionality
- Dynamic plugin loading and management

### 2. **Simplified Configuration**
- Consistent configuration format across all agents
- Environment variables only for sensitive data
- Clear separation of concerns

### 3. **Enhanced Security**
- Plugin sandboxing with resource limits
- Restricted filesystem and network access
- Service allowlists for security-critical operations

### 4. **Better Developer Experience**
- Python plugins are easier to develop and debug
- JSON communication protocol is simple and testable
- Comprehensive examples and documentation

### 5. **Operational Simplicity**
- Clean Docker setup without unnecessary tools
- Standardized logging and metrics
- Consistent health check endpoints

## 🧪 **Testing**

### Plugin Testing

```bash
# Test CPU monitor plugin
echo '{"action": "get_info"}' | python examples/plugins/cpu-monitor/cpu_monitor.py

# Test service restart plugin
echo '{"action": "execute_action", "action_request": {"id": "test", "parameters": {"service_name": "nginx"}}}' | python examples/plugins/service-restart/service_restart.py
```

### Agent Testing

```bash
# Set required environment variable
export STAVILY_AGENT_TOKEN="test-token"

# Start agents
docker-compose up -d

# Check health
curl http://localhost:8080/health  # sensor agent
curl http://localhost:8081/health  # action agent

# Check metrics
curl http://localhost:9090/metrics  # sensor agent
curl http://localhost:9091/metrics  # action agent
```

## 📚 **Documentation Updates**

- ✅ **CONFIGURATION_GUIDE.md**: Added Python plugin architecture section
- ✅ **examples/README.md**: Comprehensive guide for examples
- ✅ **examples/env.example**: Environment variables template
- ✅ **Plugin documentation**: Individual plugin configurations and usage

## 🔄 **Migration Guide**

### For Existing Deployments

1. **Update configuration files**:
   ```bash
   cp examples/configs/sensor-agent.yaml sensor-agent/configs/production.yaml
   cp examples/configs/action-agent.yaml action-agent/configs/production.yaml
   ```

2. **Set environment variables**:
   ```bash
   export STAVILY_AGENT_TOKEN="your-actual-token"
   ```

3. **Update Docker Compose**:
   - Remove old environment variables
   - Use new volume mappings
   - Remove development tools if not needed

4. **Install Python dependencies**:
   ```bash
   pip install -r examples/plugins/*/requirements.txt
   ```

### For Plugin Developers

1. **Convert Go plugins to Python**:
   - Implement JSON communication protocol
   - Create `plugin.yaml` metadata file
   - Add `requirements.txt` for dependencies

2. **Follow new plugin structure**:
   - Use standardized plugin interface
   - Implement proper error handling
   - Add resource limit considerations

## ✅ **Verification Checklist**

- [ ] Configuration files follow standard format
- [ ] Environment variables contain only sensitive data
- [ ] Python plugins implement correct interface
- [ ] Docker Compose files are simplified
- [ ] Health and metrics endpoints are accessible
- [ ] Plugin examples work correctly
- [ ] Documentation is updated and accurate

---

**Status**: ✅ **COMPLETED**

All refactoring objectives have been successfully implemented. The agents now support Python plugins with standardized configuration and simplified deployment. 