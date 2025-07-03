#!/bin/bash

# Stavily Agent Setup for localhost:8000 Orchestrator
# This script sets up both Action and Sensor agents to connect to localhost:8000

set -euo pipefail

# Configuration
ORCHESTRATOR_URL="http://localhost:8000"
BASE_DIR="/opt/stavily"
HOSTNAME=$(hostname)
ORG_ID="${STAVILY_ORG_ID:-dev-org}"
ENVIRONMENT="${STAVILY_ENV:-production}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root for directory creation
check_permissions() {
    if [[ $EUID -ne 0 && ! -w /opt ]]; then
        log_error "This script needs to create directories in /opt/stavily"
        log_error "Please run as root or ensure /opt is writable"
        exit 1
    fi
}

# Create directory structure for an agent
create_agent_directories() {
    local agent_type=$1
    local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
    
    log "Creating directory structure for ${agent_type} agent: ${agent_dir}"
    
    # Create main directories
    mkdir -p "${agent_dir}"/{config/{certificates,plugins},data/plugins,logs/{audit},tmp/workdir,cache/plugins}
    
    # Set proper permissions
    if command -v useradd >/dev/null 2>&1; then
        if ! id stavily >/dev/null 2>&1; then
            log "Creating stavily user"
            useradd -r -s /bin/false -d "${BASE_DIR}" stavily || true
        fi
        chown -R stavily:stavily "${agent_dir}" || log_warning "Could not change ownership to stavily user"
    fi
    
    # Set directory permissions
    chmod 750 "${agent_dir}"
    chmod 700 "${agent_dir}/config/certificates"
    chmod 755 "${agent_dir}/logs"
    chmod 755 "${agent_dir}/tmp/workdir"
    
    log_success "Directory structure created for ${agent_type} agent"
}

# Generate configuration file from template
generate_config() {
    local agent_type=$1
    local template_file="configs/${agent_type}-agent-localhost.yaml"
    local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
    local config_file="${agent_dir}/config/agent.yaml"
    
    log "Generating configuration for ${agent_type} agent"
    
    if [[ ! -f "$template_file" ]]; then
        log_error "Template file not found: $template_file"
        return 1
    fi
    
    # Replace template variables
    sed -e "s/\${HOSTNAME}/${HOSTNAME}/g" \
        -e "s/your-org-id/${ORG_ID}/g" \
        -e "s/production/${ENVIRONMENT}/g" \
        "$template_file" > "$config_file"
    
    chmod 640 "$config_file"
    
    log_success "Configuration generated: $config_file"
}

# Create dummy JWT token for development
create_dev_jwt_token() {
    local agent_type=$1
    local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
    local jwt_file="${agent_dir}/config/certificates/agent.jwt"
    
    log "Creating development JWT token for ${agent_type} agent"
    
    # Create a development JWT token (base64 encoded JSON for now)
    # In production, this would be a proper signed JWT from the orchestrator
    local jwt_payload='{
        "sub": "'${agent_type}'-'${HOSTNAME}'-001",
        "iss": "stavily-orchestrator",
        "aud": "stavily-agents",
        "exp": '$(date -d "+24 hours" +%s)',
        "iat": '$(date +%s)',
        "agent_type": "'${agent_type}'",
        "organization_id": "'${ORG_ID}'",
        "permissions": ["agent:register", "agent:heartbeat", "plugin:execute"]
    }'
    
    # Create a development token (not cryptographically secure)
    echo "dev.$(echo -n "$jwt_payload" | base64 -w 0).dev" > "$jwt_file"
    chmod 600 "$jwt_file"
    
    log_success "Development JWT token created: $jwt_file"
    log_warning "This is a development token. Replace with proper JWT in production!"
}

# Create self-signed certificates for localhost
create_dev_certificates() {
    local agent_type=$1
    local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
    local cert_dir="${agent_dir}/config/certificates"
    
    log "Creating development certificates for ${agent_type} agent"
    
    # Create CA certificate (self-signed)
    if [[ ! -f "${cert_dir}/ca.crt" ]]; then
        openssl req -new -x509 -days 365 -nodes \
            -out "${cert_dir}/ca.crt" \
            -keyout "${cert_dir}/ca.key" \
            -subj "/C=US/ST=Dev/L=Localhost/O=Stavily/OU=Development/CN=Stavily-CA" \
            2>/dev/null || log_warning "Could not create CA certificate (openssl not available)"
    fi
    
    # Create agent certificate
    if [[ ! -f "${cert_dir}/agent.crt" ]] && command -v openssl >/dev/null 2>&1; then
        # Generate private key
        openssl genrsa -out "${cert_dir}/agent.key" 2048 2>/dev/null
        
        # Generate certificate signing request
        openssl req -new \
            -key "${cert_dir}/agent.key" \
            -out "${cert_dir}/agent.csr" \
            -subj "/C=US/ST=Dev/L=Localhost/O=Stavily/OU=Agents/CN=${agent_type}-${HOSTNAME}-001" \
            2>/dev/null
        
        # Sign the certificate
        openssl x509 -req -in "${cert_dir}/agent.csr" \
            -CA "${cert_dir}/ca.crt" \
            -CAkey "${cert_dir}/ca.key" \
            -CAcreateserial \
            -out "${cert_dir}/agent.crt" \
            -days 365 \
            -extensions v3_req 2>/dev/null
        
        # Clean up CSR
        rm -f "${cert_dir}/agent.csr"
        
        # Set permissions
        chmod 600 "${cert_dir}"/*.key
        chmod 644 "${cert_dir}"/*.crt
        
        log_success "Development certificates created for ${agent_type} agent"
    else
        log_warning "Skipping certificate creation (openssl not available or certificates exist)"
    fi
}

# Test orchestrator connectivity
test_orchestrator_connection() {
    log "Testing connection to orchestrator at ${ORCHESTRATOR_URL}"
    
    if command -v curl >/dev/null 2>&1; then
        if curl -s -f "${ORCHESTRATOR_URL}/health" >/dev/null; then
            log_success "Orchestrator is reachable at ${ORCHESTRATOR_URL}"
            return 0
        else
            log_error "Cannot reach orchestrator at ${ORCHESTRATOR_URL}"
            log_error "Make sure the orchestrator is running: cd 02-Orchestrator && python main.py"
            return 1
        fi
    else
        log_warning "curl not available, skipping connectivity test"
        return 0
    fi
}

# Create systemd service files
create_systemd_service() {
    local agent_type=$1
    local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
    local service_file="/etc/systemd/system/stavily-${agent_type}-agent.service"
    
    if [[ ! -d /etc/systemd/system ]]; then
        log_warning "systemd not available, skipping service creation"
        return 0
    fi
    
    log "Creating systemd service for ${agent_type} agent"
    
    cat > "$service_file" << EOF
[Unit]
Description=Stavily ${agent_type^} Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=stavily
Group=stavily
WorkingDirectory=${agent_dir}
ExecStart=/usr/local/bin/stavily-${agent_type}-agent --config=${agent_dir}/config/agent.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=stavily-${agent_type}-agent

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=${agent_dir}

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "$service_file"
    log_success "Systemd service created: $service_file"
}

# Main setup function
setup_agent() {
    local agent_type=$1
    
    log "Setting up ${agent_type} agent for ${HOSTNAME}"
    
    create_agent_directories "$agent_type"
    generate_config "$agent_type"
    create_dev_jwt_token "$agent_type"
    create_dev_certificates "$agent_type"
    create_systemd_service "$agent_type"
    
    log_success "${agent_type^} agent setup completed"
}

# Main script
main() {
    log "Starting Stavily Agent Setup for localhost:8000"
    log "Hostname: ${HOSTNAME}"
    log "Organization ID: ${ORG_ID}"
    log "Environment: ${ENVIRONMENT}"
    log "Base Directory: ${BASE_DIR}"
    
    check_permissions
    
    # Test orchestrator connection first
    if ! test_orchestrator_connection; then
        log_error "Setup will continue, but agents won't be able to connect until orchestrator is running"
    fi
    
    # Setup both agents
    setup_agent "sensor"
    setup_agent "action"
    
    # Print summary
    echo
    log_success "Agent setup completed successfully!"
    echo
    echo "📋 Next Steps:"
    echo "1. Start the orchestrator: cd 02-Orchestrator && python main.py"
    echo "2. Start sensor agent: sudo systemctl start stavily-sensor-agent"
    echo "3. Start action agent: sudo systemctl start stavily-action-agent"
    echo "4. Check agent status: sudo systemctl status stavily-*-agent"
    echo
    echo "🔍 Validation Commands:"
    echo "• Test sensor agent health: curl http://localhost:8080/health"
    echo "• Test action agent health: curl http://localhost:8081/health"
    echo "• Check agent metrics: curl http://localhost:9090/metrics (sensor)"
    echo "• Check agent metrics: curl http://localhost:9091/metrics (action)"
    echo "• View logs: sudo journalctl -u stavily-sensor-agent -f"
    echo
    echo "📁 Agent Directories:"
    echo "• Sensor Agent: ${BASE_DIR}/agent-sensor-${HOSTNAME}-001"
    echo "• Action Agent: ${BASE_DIR}/agent-action-${HOSTNAME}-001"
    echo
    echo "⚠️  Remember to replace development JWT tokens with proper ones in production!"
}

# Script options
case "${1:-setup}" in
    "setup")
        main
        ;;
    "test")
        test_orchestrator_connection
        ;;
    "clean")
        log "Cleaning up agent directories..."
        rm -rf "${BASE_DIR}/agent-sensor-${HOSTNAME}-001" "${BASE_DIR}/agent-action-${HOSTNAME}-001"
        rm -f /etc/systemd/system/stavily-*-agent.service
        systemctl daemon-reload 2>/dev/null || true
        log_success "Cleanup completed"
        ;;
    *)
        echo "Usage: $0 [setup|test|clean]"
        echo "  setup: Set up both agents (default)"
        echo "  test:  Test orchestrator connectivity"
        echo "  clean: Remove agent directories and services"
        exit 1
        ;;
esac 