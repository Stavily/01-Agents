# Enhanced Stavily Agent

A Go implementation of the Stavily Agent that follows the [AGENT_USE.md](../../../02-Orchestrator/AGENT_USE.md) specification.

## Features

This enhanced agent implements the complete AGENT_USE.md workflow:

- ✅ **POST Heartbeats** - Regular heartbeat signals to the orchestrator
- ✅ **GET Instructions** - Polls for pending instructions
- ✅ **PUT Instruction Updates** - Updates instruction status during execution
- ✅ **POST Instruction Results** - Submits final execution results

### Key Capabilities

- **Base Folder Configuration**: All agent data (logs, plugins, cache) is organized under a configurable base folder
- **Server Address Configuration**: Configurable orchestrator endpoint via `api.base_url`
- **Heartbeat Management**: Automatic heartbeat sending at configurable intervals
- **Instruction Processing**: Complete instruction lifecycle management
- **Execution Logging**: Detailed execution logs sent to orchestrator
- **Error Handling**: Comprehensive error handling with retry logic
- **Graceful Shutdown**: Clean shutdown with proper resource cleanup

## Configuration

The agent uses a YAML configuration file with the following key sections:

### Agent Configuration
```yaml
agent:
  id: "enhanced-agent-001"
  base_folder: "./agent-data"  # Base folder for all agent data
  heartbeat: "30s"             # Heartbeat interval
  poll_interval: "10s"         # Instruction polling interval
```

### API Configuration
```yaml
api:
  base_url: "https://orchestrator.stavily.com"  # Server address
  agents_endpoint: "/agents/v1"
  timeout: "30s"
```

### Security Configuration
```yaml
security:
  auth:
    method: "jwt"
    token_file: "agent.jwt"  # Relative to {base_folder}/config/certificates/
```

## Quick Start

### 1. Build the Agent

```bash
make build
```

### 2. Set Up Development Environment

```bash
make dev-setup
```

This creates the necessary directory structure:
```
agent-data/
├── logs/
├── plugins/
├── cache/
└── config/
    └── certificates/
```

### 3. Configure the Agent

Edit the configuration file:
```bash
cp ../configs/enhanced-agent.yaml ./config.yaml
# Edit config.yaml with your settings
```

### 4. Validate Configuration

```bash
make validate
```

### 5. Run the Agent

```bash
# Development mode with debug logging
make dev-run

# Or production mode
make run-prod
```

## API Workflow

The agent implements the following workflow as specified in AGENT_USE.md:

### 1. Heartbeat Loop
```
POST /agents/v1/{agent_id}/heartbeat
```
- Sends heartbeat every 30 seconds (configurable)
- Indicates agent health and availability

### 2. Instruction Polling Loop
```
GET /agents/v1/{agent_id}/instructions
```
- Polls for pending instructions every 10 seconds (configurable)
- Server responds with instruction or "no pending instructions"

### 3. Instruction Processing
When an instruction is received:

1. **Update Status to Executing**:
   ```
   PUT /agents/v1/{agent_id}/instructions/{instruction_id}
   {
     "status": "executing",
     "execution_log": ["Started plugin execution"]
   }
   ```

2. **Execute the Plugin** (simulated in current implementation)

3. **Submit Final Result**:
   ```
   POST /agents/v1/{agent_id}/instructions/{instruction_id}/result
   {
     "status": "completed",
     "result": {...},
     "execution_log": ["Task completed successfully"]
   }
   ```

## Directory Structure

```
enhanced-agent/
├── main.go                 # Main application entry point
├── go.mod                  # Go module definition
├── Makefile               # Build and development tasks
├── README.md              # This file
├── config.yaml            # Configuration file (created by dev-setup)
└── agent-data/            # Base folder (created by dev-setup)
    ├── logs/              # Log files
    │   ├── enhanced-agent.log
    │   └── audit/
    ├── plugins/           # Plugin directory
    ├── cache/            # Cache directory
    └── config/           # Configuration files
        └── certificates/ # JWT tokens and certificates
```

## Development

### Available Make Targets

```bash
make help          # Show all available targets
make build         # Build the binary
make run           # Run with debug logging
make validate      # Validate configuration
make test          # Run tests
make clean         # Clean build artifacts
make fmt           # Format code
make lint          # Run linter
```

### Configuration Validation

The agent validates all configuration on startup:

```bash
./enhanced-agent validate --config config.yaml
```

Output example:
```
Configuration is valid
Agent ID: enhanced-agent-001
Agent Type: action
Tenant ID: stavily-tenant
Base Folder: ./agent-data
API Base URL: https://orchestrator.stavily.com
Heartbeat Interval: 30s
Poll Interval: 10s
```

### Health Check

When running, the agent exposes a health endpoint:

```bash
curl http://localhost:8081/health
```

### Metrics

Metrics are available at:

```bash
curl http://localhost:9091/metrics
```

## Logging

The agent uses structured JSON logging by default. Logs are written to:
- `{base_folder}/logs/enhanced-agent.log` - Main application logs
- `{base_folder}/logs/audit/enhanced-agent-audit.log` - Audit logs

Log level can be configured in the config file:
```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json" # json, text
```

## Error Handling

The agent implements comprehensive error handling:

- **Network Errors**: Automatic retry with exponential backoff
- **API Errors**: Proper HTTP status code handling
- **Timeout Errors**: Configurable timeouts for all operations
- **Plugin Errors**: Graceful error reporting to orchestrator

## Security

- **JWT Authentication**: Secure API key management
- **TLS Support**: Optional TLS encryption
- **Sandbox Mode**: Configurable resource limits
- **Audit Logging**: Complete audit trail

## Troubleshooting

### Common Issues

1. **Configuration Validation Errors**
   ```bash
   make validate
   ```

2. **Missing Directories**
   ```bash
   make dev-setup
   ```

3. **API Connection Issues**
   - Check `api.base_url` in configuration
   - Verify JWT token in `{base_folder}/config/certificates/agent.jwt`
   - Check network connectivity

4. **Permission Issues**
   ```bash
   chmod 755 agent-data
   chmod -R 644 agent-data/config/certificates/
   ```

### Debug Mode

Run with debug logging:
```bash
./enhanced-agent --config config.yaml --debug
```

### Logs

Check the logs for detailed information:
```bash
tail -f agent-data/logs/enhanced-agent.log
```

## Contributing

1. Follow Go best practices
2. Add tests for new functionality
3. Update documentation
4. Run linting: `make lint`
5. Ensure all tests pass: `make test`

## License

This project is part of the Stavily Agent system. 