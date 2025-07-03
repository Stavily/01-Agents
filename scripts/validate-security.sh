#!/bin/bash

# Stavily Agent Security Validation Script
# Validates security configuration for localhost deployment

set -euo pipefail

# Configuration
HOSTNAME="${HOSTNAME:-$(hostname)}"
BASE_DIR="/opt/stavily"
CERT_DIR="/opt/stavily/certs"

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
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

check_result() {
    local result=$1
    local success_msg=$2
    local error_msg=$3
    
    if [[ $result -eq 0 ]]; then
        log_success "$success_msg"
        return 0
    else
        log_error "$error_msg"
        return 1
    fi
}

# Check if running as appropriate user
check_user_permissions() {
    log "Checking user permissions"
    
    local errors=0
    
    # Check if running as root for setup
    if [[ $EUID -eq 0 ]]; then
        log_success "Running as root (required for validation)"
    else
        # Check if user has access to agent directories
        if [[ -r "$BASE_DIR" ]]; then
            log_success "User has read access to $BASE_DIR"
        else
            log_error "User lacks access to $BASE_DIR"
            ((errors++))
        fi
    fi
    
    return $errors
}

# Validate agent directory structure
validate_directory_structure() {
    log "Validating directory structure"
    
    local errors=0
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        
        if [[ -d "$agent_dir" ]]; then
            log_success "Agent directory exists: $agent_dir"
            
            # Check required subdirectories
            for subdir in "config/certificates" "data/plugins" "logs/audit" "tmp/workdir" "cache/plugins"; do
                if [[ -d "$agent_dir/$subdir" ]]; then
                    log_success "  Subdirectory exists: $subdir"
                else
                    log_error "  Missing subdirectory: $subdir"
                    ((errors++))
                fi
            done
            
            # Check permissions
            local perms=$(stat -c "%a" "$agent_dir" 2>/dev/null || echo "000")
            if [[ "$perms" == "750" ]] || [[ "$perms" == "755" ]]; then
                log_success "  Directory permissions are secure: $perms"
            else
                log_warning "  Directory permissions: $perms (should be 750 or 755)"
            fi
            
        else
            log_error "Agent directory missing: $agent_dir"
            ((errors++))
        fi
    done
    
    return $errors
}

# Validate TLS certificates
validate_certificates() {
    log "Validating TLS certificates"
    
    local errors=0
    
    # Check if OpenSSL is available
    if ! command -v openssl >/dev/null 2>&1; then
        log_error "OpenSSL not available for certificate validation"
        return 1
    fi
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        local cert_dir="${agent_dir}/config/certificates"
        
        if [[ -d "$cert_dir" ]]; then
            log "Checking certificates for $agent_type agent"
            
            # Check CA certificate
            if [[ -f "$cert_dir/ca.crt" ]]; then
                if openssl x509 -in "$cert_dir/ca.crt" -noout -checkend 86400 >/dev/null 2>&1; then
                    log_success "  CA certificate is valid and not expiring soon"
                else
                    log_error "  CA certificate is invalid or expiring soon"
                    ((errors++))
                fi
            else
                log_error "  CA certificate missing: $cert_dir/ca.crt"
                ((errors++))
            fi
            
            # Check agent certificate
            if [[ -f "$cert_dir/agent.crt" ]]; then
                if openssl x509 -in "$cert_dir/agent.crt" -noout -checkend 86400 >/dev/null 2>&1; then
                    log_success "  Agent certificate is valid and not expiring soon"
                    
                    # Verify against CA
                    if [[ -f "$cert_dir/ca.crt" ]]; then
                        if openssl verify -CAfile "$cert_dir/ca.crt" "$cert_dir/agent.crt" >/dev/null 2>&1; then
                            log_success "  Agent certificate is properly signed by CA"
                        else
                            log_error "  Agent certificate verification failed"
                            ((errors++))
                        fi
                    fi
                else
                    log_error "  Agent certificate is invalid or expiring soon"
                    ((errors++))
                fi
            else
                log_error "  Agent certificate missing: $cert_dir/agent.crt"
                ((errors++))
            fi
            
            # Check private key
            if [[ -f "$cert_dir/agent.key" ]]; then
                local key_perms=$(stat -c "%a" "$cert_dir/agent.key" 2>/dev/null || echo "000")
                if [[ "$key_perms" == "600" ]]; then
                    log_success "  Private key permissions are secure: $key_perms"
                else
                    log_error "  Private key permissions are insecure: $key_perms (should be 600)"
                    ((errors++))
                fi
                
                # Verify key matches certificate
                if [[ -f "$cert_dir/agent.crt" ]]; then
                    local cert_pubkey=$(openssl x509 -in "$cert_dir/agent.crt" -noout -pubkey 2>/dev/null)
                    local key_pubkey=$(openssl rsa -in "$cert_dir/agent.key" -pubout 2>/dev/null)
                    
                    if [[ "$cert_pubkey" == "$key_pubkey" ]]; then
                        log_success "  Private key matches certificate"
                    else
                        log_error "  Private key does not match certificate"
                        ((errors++))
                    fi
                fi
            else
                log_error "  Private key missing: $cert_dir/agent.key"
                ((errors++))
            fi
            
        else
            log_error "Certificate directory missing: $cert_dir"
            ((errors++))
        fi
    done
    
    return $errors
}

# Validate JWT tokens
validate_jwt_tokens() {
    log "Validating JWT tokens"
    
    local errors=0
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        local jwt_file="${agent_dir}/config/certificates/agent.jwt"
        
        if [[ -f "$jwt_file" ]]; then
            log_success "JWT token file exists for $agent_type agent"
            
            # Check file permissions
            local jwt_perms=$(stat -c "%a" "$jwt_file" 2>/dev/null || echo "000")
            if [[ "$jwt_perms" == "600" ]]; then
                log_success "  JWT token permissions are secure: $jwt_perms"
            else
                log_error "  JWT token permissions are insecure: $jwt_perms (should be 600)"
                ((errors++))
            fi
            
            # Check file is not empty
            if [[ -s "$jwt_file" ]]; then
                log_success "  JWT token file is not empty"
            else
                log_error "  JWT token file is empty"
                ((errors++))
            fi
            
            # Basic JWT structure check (3 parts separated by dots)
            local token_content=$(cat "$jwt_file" 2>/dev/null | tr -d '\n\r')
            if [[ "$token_content" =~ ^[^.]+\.[^.]+\.[^.]*$ ]]; then
                log_success "  JWT token has valid structure"
            else
                log_warning "  JWT token structure is non-standard (development token?)"
            fi
            
        else
            log_error "JWT token missing for $agent_type agent: $jwt_file"
            ((errors++))
        fi
    done
    
    return $errors
}

# Validate agent configurations
validate_agent_configs() {
    log "Validating agent configurations"
    
    local errors=0
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        local config_file="${agent_dir}/config/agent.yaml"
        
        if [[ -f "$config_file" ]]; then
            log_success "Configuration file exists for $agent_type agent"
            
            # Check basic YAML structure
            if command -v python3 >/dev/null 2>&1; then
                if python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null; then
                    log_success "  Configuration file is valid YAML"
                else
                    log_error "  Configuration file is invalid YAML"
                    ((errors++))
                fi
            fi
            
            # Check security settings
            if grep -q "tls:" "$config_file" && grep -q "enabled: true" "$config_file"; then
                log_success "  TLS is enabled in configuration"
            else
                log_warning "  TLS configuration not found or disabled"
            fi
            
            if grep -q "sandbox:" "$config_file" && grep -q "enabled: true" "$config_file"; then
                log_success "  Sandbox is enabled in configuration"
            else
                log_warning "  Sandbox configuration not found or disabled"
            fi
            
            if grep -q "audit:" "$config_file" && grep -q "enabled: true" "$config_file"; then
                log_success "  Audit logging is enabled in configuration"
            else
                log_warning "  Audit logging not found or disabled"
            fi
            
        else
            log_error "Configuration file missing for $agent_type agent: $config_file"
            ((errors++))
        fi
    done
    
    return $errors
}

# Test network connectivity
test_connectivity() {
    log "Testing network connectivity"
    
    local errors=0
    
    # Test orchestrator connectivity
    if command -v curl >/dev/null 2>&1; then
        if curl -s -f --connect-timeout 5 "http://localhost:8000/health" >/dev/null 2>&1; then
            log_success "Orchestrator is reachable at localhost:8000"
        else
            log_warning "Orchestrator not reachable at localhost:8000 (may not be running)"
        fi
    else
        log_warning "curl not available for connectivity testing"
    fi
    
    # Test agent health endpoints
    for port in 8080 8081; do
        if nc -z localhost $port 2>/dev/null; then
            log_success "Port $port is reachable (agent health endpoint)"
        else
            log_warning "Port $port is not reachable (agent may not be running)"
        fi
    done
    
    return $errors
}

# Check system security
check_system_security() {
    log "Checking system security"
    
    local errors=0
    
    # Check if stavily user exists
    if id stavily >/dev/null 2>&1; then
        log_success "Stavily user exists"
        
        # Check user shell
        local user_shell=$(getent passwd stavily | cut -d: -f7)
        if [[ "$user_shell" == "/bin/false" ]] || [[ "$user_shell" == "/usr/sbin/nologin" ]]; then
            log_success "Stavily user has secure shell: $user_shell"
        else
            log_warning "Stavily user shell may be insecure: $user_shell"
        fi
    else
        log_warning "Stavily user does not exist (agents may run as current user)"
    fi
    
    # Check firewall status
    if command -v ufw >/dev/null 2>&1; then
        if ufw status | grep -q "Status: active"; then
            log_success "UFW firewall is active"
        else
            log_warning "UFW firewall is not active"
        fi
    elif command -v firewall-cmd >/dev/null 2>&1; then
        if firewall-cmd --state 2>/dev/null | grep -q "running"; then
            log_success "Firewalld is running"
        else
            log_warning "Firewalld is not running"
        fi
    else
        log_warning "No firewall detected (ufw/firewalld)"
    fi
    
    return $errors
}

# Generate security report
generate_security_report() {
    local total_errors=$1
    
    echo
    if [[ $total_errors -eq 0 ]]; then
        log_success "Security validation completed successfully!"
        echo
        echo "🛡️  Security Status: GOOD"
        echo "All security checks passed."
    else
        log_error "Security validation completed with $total_errors error(s)"
        echo
        echo "🚨 Security Status: NEEDS ATTENTION"
        echo "Please review and fix the errors above."
    fi
    
    echo
    echo "📋 Security Checklist:"
    echo "• TLS certificates are properly configured and valid"
    echo "• JWT tokens are securely stored with correct permissions"
    echo "• Agent directories have appropriate access controls"
    echo "• Sandbox and audit logging are enabled"
    echo "• Network connectivity to orchestrator is working"
    echo
    echo "🔍 Recommended Actions:"
    echo "• Regularly rotate JWT tokens and certificates"
    echo "• Monitor audit logs for suspicious activity"
    echo "• Keep agent configurations updated"
    echo "• Test backup and recovery procedures"
    echo
    echo "📖 Documentation:"
    echo "• Configuration Guide: 01-Agents/configs/README.md"
    echo "• Security Settings: Configuration files in agent directories"
}

# Main validation function
main() {
    log "Starting Stavily Agent Security Validation"
    log "Hostname: $HOSTNAME"
    log "Base Directory: $BASE_DIR"
    
    local total_errors=0
    
    echo "==================== USER PERMISSIONS ===================="
    check_user_permissions || ((total_errors += $?))
    
    echo "==================== DIRECTORY STRUCTURE =================="
    validate_directory_structure || ((total_errors += $?))
    
    echo "==================== TLS CERTIFICATES ===================="
    validate_certificates || ((total_errors += $?))
    
    echo "==================== JWT TOKENS ==========================="
    validate_jwt_tokens || ((total_errors += $?))
    
    echo "==================== AGENT CONFIGURATIONS ================="
    validate_agent_configs || ((total_errors += $?))
    
    echo "==================== NETWORK CONNECTIVITY ================="
    test_connectivity || ((total_errors += $?))
    
    echo "==================== SYSTEM SECURITY ======================"
    check_system_security || ((total_errors += $?))
    
    echo "==================== SECURITY REPORT ======================"
    generate_security_report $total_errors
    
    return $total_errors
}

# Script options
case "${1:-validate}" in
    "validate")
        main
        ;;
    "certs")
        validate_certificates
        ;;
    "tokens")
        validate_jwt_tokens
        ;;
    "config")
        validate_agent_configs
        ;;
    "connectivity")
        test_connectivity
        ;;
    *)
        echo "Usage: $0 [validate|certs|tokens|config|connectivity]"
        echo "  validate:     Run full security validation (default)"
        echo "  certs:        Validate TLS certificates only"
        echo "  tokens:       Validate JWT tokens only" 
        echo "  config:       Validate agent configurations only"
        echo "  connectivity: Test network connectivity only"
        exit 1
        ;;
esac 