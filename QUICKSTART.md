# Stavily Agents Quick Start

Get up and running with Stavily agents in under 5 minutes.

## Prerequisites

- Docker and Docker Compose
- Valid Stavily account (sign up at https://app.stavily.com)

## 1. Quick Start with Docker Compose

```bash
# Clone or download the agents
git clone https://github.com/stavily/agents.git
cd agents

# Start both agents
docker-compose up -d

# Check agent status
docker-compose ps
```

## 2. Access Points

Once running, you can access:

- **Sensor Agent Health**: http://localhost:8080/health
- **Action Agent Health**: http://localhost:8081/health

## 3. Configuration

### Environment Variables

Key environment variables for quick configuration:

```bash
# Sensor Agent
export STAVILY_SENSOR_AGENT_ID=sensor-quickstart-001
export STAVILY_SENSOR_API_BASE_URL=https://agents.stavily.com
export STAVILY_SENSOR_LOGGING_LEVEL=info

# Action Agent  
export STAVILY_ACTION_AGENT_ID=action-quickstart-001
export STAVILY_ACTION_API_BASE_URL=https://agents.stavily.com
export STAVILY_ACTION_LOGGING_LEVEL=info
```

### Quick Configuration Files

Create minimal configuration files:

**Sensor Agent** (`sensor-agent/configs/dev.yaml`):
```yaml
agent:
  id: "sensor-quickstart-001"
  name: "Quick Start Sensor"
  type: "sensor"
  organization_id: "your-org-id"

api:
  base_url: "https://agents.stavily.com"
  
logging:
  level: "info"
  format: "json"
```

**Action Agent** (`action-agent/configs/dev.yaml`):
```yaml
agent:
  id: "action-quickstart-001"
  name: "Quick Start Action"
  type: "action"
  organization_id: "your-org-id"

api:
  base_url: "https://agents.stavily.com"
  
logging:
  level: "info"
  format: "json"
```

## 4. Build from Source

```bash
# Build all agents
make build

# Run sensor agent
./bin/sensor-agent --config sensor-agent/configs/dev.yaml

# Run action agent (in another terminal)
./bin/action-agent --config action-agent/configs/dev.yaml
```

## 5. Verify Installation

```bash
# Check agent health
curl http://localhost:8080/health  # Sensor agent
curl http://localhost:8081/health  # Action agent

# View logs
docker-compose logs sensor-agent
docker-compose logs action-agent
```

## 6. Next Steps

1. **Get API Credentials**: Visit https://app.stavily.com to get your organization ID and API credentials
2. **Configure Authentication**: Set up certificate-based authentication for production
3. **Install Plugins**: Add trigger and action plugins for your use case
4. **Create Workflows**: Use the web interface to create automation workflows

## Common Issues

### Connection Issues
- Ensure outbound HTTPS access to `agents.stavily.com`
- Check firewall settings
- Verify DNS resolution

### Configuration Issues
- Validate YAML syntax in configuration files
- Check environment variable names and values
- Ensure organization ID is correct

### Docker Issues
- Verify Docker daemon is running
- Check port availability (8080, 8081)
- Ensure sufficient disk space for images

## Documentation

- **Full Documentation**: See [README.md](README.md)
- **Configuration Guide**: See [docs/CONFIGURATION_GUIDE.md](docs/CONFIGURATION_GUIDE.md)  
- **Deployment Guide**: See [docs/DEPLOYMENT_GUIDE.md](docs/DEPLOYMENT_GUIDE.md)

## Support

- **Issues**: Create an issue in the repository
- **Community**: Join our community forum
- **Enterprise Support**: Contact support@stavily.com 