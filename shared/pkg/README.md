# Stavily Agent Plugin System

This package provides comprehensive plugin download and execution capabilities for Stavily agents. The system enables agents to dynamically install plugins from Git repositories and execute them based on instructions received through the polling mechanism.

## 📋 Overview

The plugin system consists of several key components:

- **Instruction Types**: Data structures for handling plugin-related instructions
- **Plugin Downloader**: Handles git clone operations and plugin installation
- **Plugin Executor**: Executes installed plugins with multiple runtime support
- **Instruction Handler**: Processes instructions from polling responses
- **Enhanced Plugin Manager**: Integrates all components with the existing plugin system

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Polling       │    │   Instruction   │    │   Plugin        │
│   Response      │───▶│   Handler       │───▶│   Downloader    │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Plugin        │
                       │   Executor      │
                       │                 │
                       └─────────────────┘
```

## 📦 Components

### 1. Instruction Types (`pkg/instruction/types.go`)

Defines the data structures for handling plugin instructions:

```go
type InstructionType string

const (
    InstructionTypePluginInstall InstructionType = "plugin_install"
    InstructionTypeExecute       InstructionType = "execute"
)

type Instruction struct {
    ID                  string                 `json:"id"`
    AgentID             string                 `json:"agent_id"`
    PluginID            string                 `json:"plugin_id"`
    InstructionType     InstructionType        `json:"instruction_type"`
    PluginConfiguration map[string]interface{} `json:"plugin_configuration"`
    InputData           map[string]interface{} `json:"input_data"`
    // ... additional fields
}
```

### 2. Plugin Downloader (`pkg/plugin/downloader.go`)

Handles plugin installation from Git repositories:

- **Git Clone Support**: Downloads plugins from public/private repositories
- **Version Control**: Supports branches, tags, and specific commit hashes
- **Validation**: Verifies plugin structure after download
- **Error Handling**: Comprehensive error handling and cleanup

**Key Features:**
- Shallow clones for faster downloads
- Timeout support for git operations
- Plugin structure verification
- Failed installation cleanup

### 3. Plugin Executor (`pkg/plugin/executor.go`)

Executes installed plugins with multi-runtime support:

- **Runtime Detection**: Automatically detects Python, Node.js, Bash, Docker, and executable plugins
- **Input Handling**: Passes structured data to plugins via JSON files or arguments
- **Environment Management**: Sets up environment variables and working directories
- **Output Parsing**: Attempts to parse structured output from plugins

**Supported Runtimes:**
- Python (`python3`)
- Node.js (`node`)
- Bash scripts (`bash`)
- Docker containers (`docker`)
- Native executables

### 4. Instruction Handler (`pkg/instruction/handler.go`)

Processes instructions from polling responses:

- **Validation**: Validates instructions before processing
- **Orchestration**: Coordinates between downloader and executor
- **Logging**: Detailed execution logging and error tracking
- **Result Generation**: Creates structured results for each instruction

### 5. Enhanced Plugin Manager (`pkg/agent/enhanced_plugin_manager.go`)

Integrates all components with the existing plugin system:

- **Unified Interface**: Single interface for all plugin operations
- **Instruction Processing**: Handles poll responses automatically
- **Direct Operations**: Supports direct plugin installation and execution
- **Status Monitoring**: Provides comprehensive status information

## 🚀 Usage

### Basic Setup

```go
import (
    "github.com/Stavily/01-Agents/shared/pkg/agent"
    "github.com/Stavily/01-Agents/shared/pkg/config"
    "go.uber.org/zap"
)

// Create configuration
cfg := &agent.EnhancedPluginConfig{
    PluginConfig: &config.PluginConfig{
        Enabled: true,
    },
    PluginBaseDir: "./plugins",
    GitTimeout:    5 * time.Minute,
    ExecTimeout:   10 * time.Minute,
}

// Create logger
logger, _ := zap.NewDevelopment()

// Create enhanced plugin manager
pluginManager, err := agent.NewEnhancedPluginManager(cfg, logger)
if err != nil {
    log.Fatal(err)
}

// Initialize
ctx := context.Background()
pluginManager.Initialize(ctx)
defer pluginManager.Shutdown(ctx)
```

### Processing Poll Responses

The system automatically handles the polling responses as specified in the user requirements:

#### No Pending Instructions
```json
{
  "instruction": null,
  "status": "no_pending_instructions",
  "next_poll_interval": 10
}
```

#### Plugin Installation Instruction
```json
{
  "instruction": {
    "id": "0aeb8570-9b0e-4dd5-b567-76261e29f0d7",
    "agent_id": "65bb7a51-1812-4a37-90d5-3e8a859ba972",
    "plugin_id": "e56434ca-9756-446e-92cd-f7545cb5b7b2",
         "instruction_type": "plugin_install",
     "plugin_configuration": {
       "plugin_url": "https://github.com/stavily/06-plugins",
       "entrypoint": "main.py"
     },
    "timeout_seconds": 300
  },
  "status": "instruction_delivered",
  "next_poll_interval": 5
}
```

#### Plugin Execution Instruction
```json
{
  "instruction": {
    "id": "26a71c76-69d9-4923-a86b-783d9796fb17",
    "agent_id": "65bb7a51-1812-4a37-90d5-3e8a859ba972",
    "plugin_id": "8024d189-0016-48cc-9f01-16d54210ac59",
    "instruction_type": "execute",
    "plugin_configuration": {
      "entrypoint": "main.py"
    },
    "input_data": {
      "task": "process_data",
      "parameters": {"key": "value"}
    },
    "timeout_seconds": 300
  },
  "status": "instruction_delivered",
  "next_poll_interval": 5
}
```

### Processing Instructions

```go
// Process poll response
result, err := pluginManager.ProcessInstruction(ctx, pollResponse)
if err != nil {
    logger.Error("Instruction processing failed", zap.Error(err))
    return
}

if result != nil {
    logger.Info("Instruction completed",
        zap.String("instruction_id", result.InstructionID),
        zap.String("status", string(result.Status)),
        zap.Duration("duration", result.Duration))
}
```

### Direct Plugin Operations

```go
// Direct plugin installation
installResult, err := pluginManager.InstallPlugin(
    ctx,
    "my-plugin",
    "https://github.com/stavily/06-plugins",
    "main",
)

// Direct plugin execution
execResult, err := pluginManager.ExecutePlugin(
    ctx,
    "my-plugin",
    "main.py",
    map[string]interface{}{
        "input": "data",
    },
)

// Check if plugin is installed
if pluginManager.IsPluginInstalled("my-plugin") {
    path := pluginManager.GetInstalledPluginPath("my-plugin")
    fmt.Printf("Plugin installed at: %s\n", path)
}
```

## 🔧 Configuration

### Plugin Configuration Structure

For plugin installation, the `plugin_configuration` should include:

```json
{
  "plugin_url": "https://github.com/stavily/06-plugins",
  "entrypoint": "main.py",      // Required: entry point file
  "version": "v1.0.0",          // Optional: specific version
  "branch": "main",             // Optional: specific branch
  "tag": "v1.0.0",             // Optional: specific tag
  "commit_hash": "abc123...",   // Optional: specific commit
  "sub_directory": "plugin/"    // Optional: subdirectory in repo
}
```

For plugin execution, the `plugin_configuration` should include:

```json
{
  "entrypoint": "main.py",                    // Required: entry point file
  "arguments": ["--verbose", "--config"],    // Optional: command arguments
  "environment": {                           // Optional: environment variables
    "PLUGIN_ENV": "production",
    "API_KEY": "secret"
  },
  "timeout_seconds": 60                      // Optional: execution timeout
}
```

### Supported Plugin Structures

The system supports various plugin structures:

#### Python Plugins
- `main.py`, `index.py`, or any `.py` file as entrypoint
- Optional `requirements.txt` for dependencies
- Input data passed via `--input input.json` argument

#### Node.js Plugins
- `main.js`, `index.js`, or any `.js`/`.mjs` file as entrypoint
- Optional `package.json` for dependencies
- Input data passed via `--input input.json` argument

#### Bash Scripts
- Any `.sh` file as entrypoint
- Input data passed via environment variables

#### Docker Plugins
- `Dockerfile` for containerized execution
- Environment variables passed to container

#### Executable Plugins
- Any executable binary
- Input data handling depends on plugin implementation

## 📝 Input Data Format

When plugins are executed, input data is provided in a structured format:

```json
{
  "input_data": {
    "task": "process_data",
    "parameters": {"key": "value"}
  },
  "context": {
    "execution_id": "exec-123",
    "workflow_id": "wf-456"
  },
  "variables": {
    "api_key": "secret-key-123"
  }
}
```

This data is:
- Written to `input.json` file in the plugin directory
- Passed via `--input input.json` argument for Python/Node.js plugins
- Available as environment variables for other runtime types

## 📊 Result Format

Instruction results follow a consistent structure:

```go
type InstructionResult struct {
    InstructionID string                 `json:"instruction_id"`
    Status        InstructionStatus      `json:"status"`
    Result        map[string]interface{} `json:"result"`
    Error         string                 `json:"error,omitempty"`
    Duration      time.Duration          `json:"duration"`
    ExecutionLog  []LogEntry             `json:"execution_log"`
    CompletedAt   time.Time              `json:"completed_at"`
}
```

## 🛠️ Error Handling

The system provides comprehensive error handling:

- **Validation Errors**: Missing required fields or invalid configurations
- **Download Errors**: Git clone failures, network issues, or authentication problems
- **Execution Errors**: Plugin runtime errors, timeouts, or crashes
- **Cleanup**: Automatic cleanup of failed installations

## 📈 Monitoring and Logging

All operations are thoroughly logged with structured logging:

```go
// Installation logging
logger.Info("Starting plugin download",
    zap.String("instruction_id", inst.ID),
    zap.String("plugin_id", inst.PluginID))

// Execution logging
logger.Info("Plugin execution completed",
    zap.String("instruction_id", inst.ID),
    zap.Bool("success", result.Success),
    zap.Duration("duration", result.Duration))
```

## 🔍 Status and Health Monitoring

```go
// Get enhanced status
status := pluginManager.GetEnhancedStatus()

// Check pending instructions
pending := pluginManager.GetPendingInstructions()

// Validate instruction support
supported := pluginManager.ValidateInstructionSupport(
    instruction.InstructionTypePluginInstall,
)
```

## 📚 Examples

See `pkg/examples/plugin_usage_example.go` for comprehensive usage examples including:

- Processing poll responses
- Direct plugin operations
- Polling loop simulation
- Error handling patterns

## 🔒 Security Considerations

- Plugin execution is isolated with configurable timeouts
- Git operations support authentication tokens and SSH keys
- Plugin validation ensures basic structure requirements
- Comprehensive logging for audit trails

## 🚧 Future Enhancements

- Plugin sandboxing and security policies
- Plugin dependency management
- Plugin versioning and updates
- Plugin marketplace integration
- Enhanced runtime support (Ruby, Go, etc.)

## 📄 Dependencies

- `go.uber.org/zap`: Structured logging
- Standard Go libraries for file operations, process execution, and JSON handling
- Git must be available in the system PATH for plugin downloads 