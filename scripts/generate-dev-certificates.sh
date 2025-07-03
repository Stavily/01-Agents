#!/bin/bash

# Generate Development TLS Certificates for Stavily Agents
# This script creates self-signed certificates for localhost development

set -euo pipefail

# Configuration
CERT_DIR="${1:-/opt/stavily/certs}"
HOSTNAME="${HOSTNAME:-$(hostname)}"
DAYS="${CERT_DAYS:-365}"

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

# Check if OpenSSL is available
check_openssl() {
    if ! command -v openssl >/dev/null 2>&1; then
        log_error "OpenSSL is not installed or not in PATH"
        exit 1
    fi
}

# Create certificate directory
create_cert_dir() {
    log "Creating certificate directory: $CERT_DIR"
    mkdir -p "$CERT_DIR"
    chmod 755 "$CERT_DIR"
}

# Generate CA certificate and key
generate_ca() {
    log "Generating Certificate Authority (CA)"
    
    # Generate CA private key
    openssl genrsa -out "$CERT_DIR/ca.key" 4096
    chmod 600 "$CERT_DIR/ca.key"
    
    # Generate CA certificate
    openssl req -new -x509 -days $DAYS -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" \
        -subj "/C=US/ST=Development/L=Localhost/O=Stavily/OU=Development/CN=Stavily-Development-CA"
    chmod 644 "$CERT_DIR/ca.crt"
    
    log_success "CA certificate generated"
}

# Generate server certificate for orchestrator
generate_server_cert() {
    log "Generating server certificate for orchestrator"
    
    # Create config file for server certificate
    cat > "$CERT_DIR/server.conf" << EOF
[req]
default_bits = 2048
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C = US
ST = Development
L = Localhost
O = Stavily
OU = Orchestrator
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = 127.0.0.1
DNS.3 = ::1
DNS.4 = ${HOSTNAME}
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

    # Generate server private key
    openssl genrsa -out "$CERT_DIR/server.key" 2048
    chmod 600 "$CERT_DIR/server.key"
    
    # Generate server certificate signing request
    openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
        -config "$CERT_DIR/server.conf"
    
    # Sign server certificate with CA
    openssl x509 -req -in "$CERT_DIR/server.csr" \
        -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
        -out "$CERT_DIR/server.crt" -days $DAYS \
        -extensions v3_req -extfile "$CERT_DIR/server.conf"
    chmod 644 "$CERT_DIR/server.crt"
    
    # Clean up CSR
    rm -f "$CERT_DIR/server.csr" "$CERT_DIR/server.conf"
    
    log_success "Server certificate generated"
}

# Generate agent certificate
generate_agent_cert() {
    local agent_type=$1
    local agent_id="${agent_type}-${HOSTNAME}-001"
    
    log "Generating ${agent_type} agent certificate"
    
    # Create config file for agent certificate
    cat > "$CERT_DIR/${agent_type}-agent.conf" << EOF
[req]
default_bits = 2048
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C = US
ST = Development
L = Localhost
O = Stavily
OU = Agents
CN = ${agent_id}

[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${agent_id}
DNS.2 = localhost
DNS.3 = ${HOSTNAME}
EOF

    # Generate agent private key
    openssl genrsa -out "$CERT_DIR/${agent_type}-agent.key" 2048
    chmod 600 "$CERT_DIR/${agent_type}-agent.key"
    
    # Generate agent certificate signing request
    openssl req -new -key "$CERT_DIR/${agent_type}-agent.key" \
        -out "$CERT_DIR/${agent_type}-agent.csr" \
        -config "$CERT_DIR/${agent_type}-agent.conf"
    
    # Sign agent certificate with CA
    openssl x509 -req -in "$CERT_DIR/${agent_type}-agent.csr" \
        -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
        -out "$CERT_DIR/${agent_type}-agent.crt" -days $DAYS \
        -extensions v3_req -extfile "$CERT_DIR/${agent_type}-agent.conf"
    chmod 644 "$CERT_DIR/${agent_type}-agent.crt"
    
    # Clean up
    rm -f "$CERT_DIR/${agent_type}-agent.csr" "$CERT_DIR/${agent_type}-agent.conf"
    
    log_success "${agent_type} agent certificate generated"
}

# Copy certificates to agent directories
copy_certs_to_agents() {
    log "Copying certificates to agent directories"
    
    for agent_type in "sensor" "action"; do
        local agent_dir="/opt/stavily/agent-${agent_type}-${HOSTNAME}-001"
        local cert_dir="${agent_dir}/config/certificates"
        
        if [[ -d "$agent_dir" ]]; then
            mkdir -p "$cert_dir"
            
            # Copy CA certificate
            cp "$CERT_DIR/ca.crt" "$cert_dir/"
            
            # Copy agent-specific certificates
            cp "$CERT_DIR/${agent_type}-agent.crt" "$cert_dir/agent.crt"
            cp "$CERT_DIR/${agent_type}-agent.key" "$cert_dir/agent.key"
            
            # Set proper permissions
            chmod 644 "$cert_dir/ca.crt" "$cert_dir/agent.crt"
            chmod 600 "$cert_dir/agent.key"
            
            # Change ownership if stavily user exists
            if id stavily >/dev/null 2>&1; then
                chown -R stavily:stavily "$cert_dir"
            fi
            
            log_success "Certificates copied to ${agent_type} agent directory"
        else
            log_warning "Agent directory not found: $agent_dir"
        fi
    done
}

# Verify certificates
verify_certificates() {
    log "Verifying certificates"
    
    # Verify CA certificate
    if openssl x509 -in "$CERT_DIR/ca.crt" -noout -text >/dev/null 2>&1; then
        log_success "CA certificate is valid"
    else
        log_error "CA certificate is invalid"
        return 1
    fi
    
    # Verify server certificate
    if openssl verify -CAfile "$CERT_DIR/ca.crt" "$CERT_DIR/server.crt" >/dev/null 2>&1; then
        log_success "Server certificate is valid"
    else
        log_error "Server certificate is invalid"
        return 1
    fi
    
    # Verify agent certificates
    for agent_type in "sensor" "action"; do
        if openssl verify -CAfile "$CERT_DIR/ca.crt" "$CERT_DIR/${agent_type}-agent.crt" >/dev/null 2>&1; then
            log_success "${agent_type} agent certificate is valid"
        else
            log_error "${agent_type} agent certificate is invalid"
            return 1
        fi
    done
}

# Display certificate information
show_certificate_info() {
    echo
    log_success "Certificate generation completed!"
    echo
    echo "📁 Certificate Directory: $CERT_DIR"
    echo
    echo "📋 Generated Certificates:"
    echo "• CA Certificate: ca.crt"
    echo "• CA Private Key: ca.key"
    echo "• Server Certificate: server.crt (for orchestrator)"
    echo "• Server Private Key: server.key"
    echo "• Sensor Agent Certificate: sensor-agent.crt"
    echo "• Sensor Agent Private Key: sensor-agent.key"
    echo "• Action Agent Certificate: action-agent.crt"
    echo "• Action Agent Private Key: action-agent.key"
    echo
    echo "🔒 Security Information:"
    echo "• All certificates valid for: $DAYS days"
    echo "• CA Subject: /C=US/ST=Development/L=Localhost/O=Stavily/OU=Development/CN=Stavily-Development-CA"
    echo "• Server Subject: /C=US/ST=Development/L=Localhost/O=Stavily/OU=Orchestrator/CN=localhost"
    echo
    echo "⚠️  Development Certificates Warning:"
    echo "These certificates are for development use only."
    echo "Do NOT use these certificates in production environments."
    echo
    echo "🔍 Verification Commands:"
    echo "• Verify CA: openssl x509 -in $CERT_DIR/ca.crt -noout -text"
    echo "• Verify Server: openssl verify -CAfile $CERT_DIR/ca.crt $CERT_DIR/server.crt"
    echo "• Test Connection: openssl s_client -connect localhost:8000 -CAfile $CERT_DIR/ca.crt"
}

# Main function
main() {
    log "Starting certificate generation for Stavily development"
    log "Certificate directory: $CERT_DIR"
    log "Hostname: $HOSTNAME"
    log "Validity period: $DAYS days"
    
    check_openssl
    create_cert_dir
    generate_ca
    generate_server_cert
    generate_agent_cert "sensor"
    generate_agent_cert "action"
    copy_certs_to_agents
    verify_certificates
    show_certificate_info
}

# Script options
case "${1:-generate}" in
    "generate")
        main
        ;;
    "verify")
        if [[ -d "$CERT_DIR" ]]; then
            verify_certificates
        else
            log_error "Certificate directory not found: $CERT_DIR"
            exit 1
        fi
        ;;
    "clean")
        log "Cleaning certificate directory: $CERT_DIR"
        rm -rf "$CERT_DIR"
        log_success "Certificate directory cleaned"
        ;;
    *)
        echo "Usage: $0 [generate|verify|clean] [cert_directory]"
        echo "  generate: Generate all certificates (default)"
        echo "  verify:   Verify existing certificates"
        echo "  clean:    Remove certificate directory"
        echo
        echo "Environment variables:"
        echo "  CERT_DAYS: Certificate validity in days (default: 365)"
        echo "  HOSTNAME:  Override hostname (default: $(hostname))"
        exit 1
        ;;
esac 