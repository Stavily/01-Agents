#!/bin/bash

# Stavily Agent-Orchestrator Connection Testing Script
# Tests the complete agent connection workflow with localhost:8000

set -euo pipefail

# Configuration
ORCHESTRATOR_URL="http://localhost:8000"
HOSTNAME="${HOSTNAME:-$(hostname)}"
BASE_DIR="/opt/stavily"
TIMEOUT="${TEST_TIMEOUT:-30}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓ PASS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠ WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗ FAIL]${NC} $1"
}

log_info() {
    echo -e "${CYAN}[ℹ INFO]${NC} $1"
}

log_test() {
    echo -e "${PURPLE}[🧪 TEST]${NC} $1"
}

# Test results tracking
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_WARNINGS=0

# Function to record test result
record_test() {
    local result=$1
    local test_name=$2
    
    ((TESTS_TOTAL++))
    
    case $result in
        "pass")
            ((TESTS_PASSED++))
            log_success "$test_name"
            ;;
        "fail")
            ((TESTS_FAILED++))
            log_error "$test_name"
            ;;
        "warn")
            ((TESTS_WARNINGS++))
            log_warning "$test_name"
            ;;
    esac
}

# Check prerequisites
check_prerequisites() {
    log_test "Checking prerequisites"
    
    local errors=0
    
    # Check if curl is available
    if command -v curl >/dev/null 2>&1; then
        record_test "pass" "curl is available"
    else
        record_test "fail" "curl is not available"
        ((errors++))
    fi
    
    # Check if jq is available for JSON parsing
    if command -v jq >/dev/null 2>&1; then
        record_test "pass" "jq is available for JSON parsing"
    else
        record_test "warn" "jq not available - JSON parsing will be limited"
    fi
    
    # Check if nc (netcat) is available for port testing
    if command -v nc >/dev/null 2>&1; then
        record_test "pass" "netcat is available for port testing"
    else
        record_test "warn" "netcat not available - port testing will be limited"
    fi
    
    return $errors
}

# Test orchestrator availability
test_orchestrator_health() {
    log_test "Testing orchestrator health"
    
    # Test basic connectivity
    if curl -s -f --connect-timeout $TIMEOUT "${ORCHESTRATOR_URL}/health" >/dev/null 2>&1; then
        record_test "pass" "Orchestrator health endpoint is accessible"
    else
        record_test "fail" "Orchestrator health endpoint not accessible at ${ORCHESTRATOR_URL}/health"
        return 1
    fi
    
    # Test health response format
    local health_response=$(curl -s --connect-timeout $TIMEOUT "${ORCHESTRATOR_URL}/health" 2>/dev/null || echo "{}")
    
    if command -v jq >/dev/null 2>&1; then
        if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
            local status=$(echo "$health_response" | jq -r '.status')
            if [[ "$status" == "healthy" ]]; then
                record_test "pass" "Orchestrator reports healthy status"
            else
                record_test "warn" "Orchestrator status: $status"
            fi
        else
            record_test "warn" "Health response format is non-standard"
        fi
    else
        if echo "$health_response" | grep -q "healthy\|status"; then
            record_test "pass" "Health response contains status information"
        else
            record_test "warn" "Health response format could not be verified"
        fi
    fi
    
    return 0
}

# Test agent registration
test_agent_registration() {
    local agent_type=$1
    log_test "Testing $agent_type agent registration"
    
    # Prepare registration payload
    local agent_id="${agent_type}-${HOSTNAME}-test-001"
    local registration_data=$(cat << EOF
{
    "id": "$agent_id",
    "name": "Test $agent_type Agent",
    "type": "$agent_type",
    "organization_id": "test-org",
    "version": "1.0.0-test",
    "hostname": "$HOSTNAME",
    "ip_address": "127.0.0.1",
    "platform": "$(uname -s | tr '[:upper:]' '[:lower:]')",
    "arch": "$(uname -m)",
    "capabilities": ["test-capability"],
    "config": {
        "environment": "test",
        "tags": ["testing"]
    }
}
EOF
)
    
    # Test registration
    local response=$(curl -s -w "%{http_code}" --connect-timeout $TIMEOUT \
        -H "Content-Type: application/json" \
        -d "$registration_data" \
        "${ORCHESTRATOR_URL}/api/v1/agents" 2>/dev/null || echo "000")
    
    local http_code="${response: -3}"
    local response_body="${response%???}"
    
    if [[ "$http_code" == "201" ]]; then
        record_test "pass" "$agent_type agent registration successful (HTTP $http_code)"
        
        # Parse response if jq is available
        if command -v jq >/dev/null 2>&1; then
            if echo "$response_body" | jq -e '.agent_id' >/dev/null 2>&1; then
                local returned_agent_id=$(echo "$response_body" | jq -r '.agent_id')
                if [[ "$returned_agent_id" == "$agent_id" ]]; then
                    record_test "pass" "Registration response contains correct agent ID"
                else
                    record_test "warn" "Agent ID mismatch in response: $returned_agent_id"
                fi
            fi
            
            if echo "$response_body" | jq -e '.api_key' >/dev/null 2>&1; then
                record_test "pass" "Registration response contains API key"
            else
                record_test "warn" "Registration response missing API key"
            fi
        fi
        
    elif [[ "$http_code" == "400" ]] && echo "$response_body" | grep -q "already exists"; then
        record_test "pass" "$agent_type agent already registered (HTTP $http_code)"
    else
        record_test "fail" "$agent_type agent registration failed (HTTP $http_code): $response_body"
        return 1
    fi
    
    # Test agent cleanup (deregistration)
    if [[ "$http_code" == "201" ]]; then
        log_test "Testing $agent_type agent deregistration"
        local delete_response=$(curl -s -w "%{http_code}" --connect-timeout $TIMEOUT \
            -X DELETE "${ORCHESTRATOR_URL}/api/v1/agents/$agent_id" 2>/dev/null || echo "000")
        
        local delete_code="${delete_response: -3}"
        
        if [[ "$delete_code" == "200" ]] || [[ "$delete_code" == "204" ]]; then
            record_test "pass" "$agent_type agent deregistration successful"
        else
            record_test "warn" "$agent_type agent deregistration returned HTTP $delete_code"
        fi
    fi
    
    return 0
}

# Test agent health endpoints
test_agent_health_endpoints() {
    log_test "Testing agent health endpoints"
    
    # Test sensor agent health endpoint (port 8080)
    if nc -z localhost 8080 2>/dev/null; then
        record_test "pass" "Sensor agent health endpoint is accessible (port 8080)"
        
        # Try to get health response
        if curl -s -f --connect-timeout 5 "http://localhost:8080/health" >/dev/null 2>&1; then
            record_test "pass" "Sensor agent health endpoint responds correctly"
        else
            record_test "warn" "Sensor agent health endpoint port is open but not responding"
        fi
    else
        record_test "warn" "Sensor agent health endpoint not accessible (agent may not be running)"
    fi
    
    # Test action agent health endpoint (port 8081)
    if nc -z localhost 8081 2>/dev/null; then
        record_test "pass" "Action agent health endpoint is accessible (port 8081)"
        
        # Try to get health response
        if curl -s -f --connect-timeout 5 "http://localhost:8081/health" >/dev/null 2>&1; then
            record_test "pass" "Action agent health endpoint responds correctly"
        else
            record_test "warn" "Action agent health endpoint port is open but not responding"
        fi
    else
        record_test "warn" "Action agent health endpoint not accessible (agent may not be running)"
    fi
}

# Test agent metrics endpoints
test_agent_metrics_endpoints() {
    log_test "Testing agent metrics endpoints"
    
    # Test sensor agent metrics endpoint (port 9090)
    if nc -z localhost 9090 2>/dev/null; then
        record_test "pass" "Sensor agent metrics endpoint is accessible (port 9090)"
        
        # Try to get metrics
        if curl -s --connect-timeout 5 "http://localhost:9090/metrics" | head -5 | grep -q "#"; then
            record_test "pass" "Sensor agent metrics endpoint returns Prometheus format"
        else
            record_test "warn" "Sensor agent metrics endpoint format could not be verified"
        fi
    else
        record_test "warn" "Sensor agent metrics endpoint not accessible (agent may not be running)"
    fi
    
    # Test action agent metrics endpoint (port 9091)
    if nc -z localhost 9091 2>/dev/null; then
        record_test "pass" "Action agent metrics endpoint is accessible (port 9091)"
        
        # Try to get metrics
        if curl -s --connect-timeout 5 "http://localhost:9091/metrics" | head -5 | grep -q "#"; then
            record_test "pass" "Action agent metrics endpoint returns Prometheus format"
        else
            record_test "warn" "Action agent metrics endpoint format could not be verified"
        fi
    else
        record_test "warn" "Action agent metrics endpoint not accessible (agent may not be running)"
    fi
}

# Test agent configuration files
test_agent_configurations() {
    log_test "Testing agent configuration files"
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        local config_file="${agent_dir}/config/agent.yaml"
        
        if [[ -f "$config_file" ]]; then
            record_test "pass" "$agent_type agent configuration file exists"
            
            # Check if configuration points to localhost:8000
            if grep -q "localhost:8000" "$config_file"; then
                record_test "pass" "$agent_type agent configured for localhost:8000"
            else
                record_test "warn" "$agent_type agent may not be configured for localhost:8000"
            fi
            
            # Check if TLS is configured
            if grep -q "tls:" "$config_file"; then
                record_test "pass" "$agent_type agent has TLS configuration"
            else
                record_test "warn" "$agent_type agent missing TLS configuration"
            fi
            
            # Check if JWT authentication is configured
            if grep -q "token_file:" "$config_file"; then
                record_test "pass" "$agent_type agent has JWT token configuration"
            else
                record_test "warn" "$agent_type agent missing JWT token configuration"
            fi
            
        else
            record_test "fail" "$agent_type agent configuration file missing: $config_file"
        fi
    done
}

# Test JWT tokens
test_jwt_tokens() {
    log_test "Testing JWT tokens"
    
    for agent_type in "sensor" "action"; do
        local agent_dir="${BASE_DIR}/agent-${agent_type}-${HOSTNAME}-001"
        local jwt_file="${agent_dir}/config/certificates/agent.jwt"
        
        if [[ -f "$jwt_file" ]]; then
            record_test "pass" "$agent_type agent JWT token file exists"
            
            # Check if token is not empty
            if [[ -s "$jwt_file" ]]; then
                record_test "pass" "$agent_type agent JWT token is not empty"
            else
                record_test "fail" "$agent_type agent JWT token file is empty"
            fi
            
            # Check file permissions
            local perms=$(stat -c "%a" "$jwt_file" 2>/dev/null || echo "000")
            if [[ "$perms" == "600" ]]; then
                record_test "pass" "$agent_type agent JWT token has secure permissions"
            else
                record_test "warn" "$agent_type agent JWT token permissions: $perms (should be 600)"
            fi
            
        else
            record_test "fail" "$agent_type agent JWT token file missing: $jwt_file"
        fi
    done
}

# Test orchestrator API endpoints
test_orchestrator_api_endpoints() {
    log_test "Testing orchestrator API endpoints"
    
    # Test agents endpoint
    local agents_response=$(curl -s -w "%{http_code}" --connect-timeout $TIMEOUT \
        "${ORCHESTRATOR_URL}/api/v1/agents" 2>/dev/null || echo "000")
    
    local agents_code="${agents_response: -3}"
    
    if [[ "$agents_code" == "200" ]]; then
        record_test "pass" "Agents API endpoint is accessible"
    else
        record_test "warn" "Agents API endpoint returned HTTP $agents_code"
    fi
    
    # Test monitoring endpoint
    local monitoring_response=$(curl -s -w "%{http_code}" --connect-timeout $TIMEOUT \
        "${ORCHESTRATOR_URL}/api/v1/monitoring/metrics" 2>/dev/null || echo "000")
    
    local monitoring_code="${monitoring_response: -3}"
    
    if [[ "$monitoring_code" == "200" ]]; then
        record_test "pass" "Monitoring API endpoint is accessible"
    else
        record_test "warn" "Monitoring API endpoint returned HTTP $monitoring_code"
    fi
}

# Generate test report
generate_test_report() {
    echo
    echo "═══════════════════════════════════════════════════════════════"
    echo "                    AGENT CONNECTION TEST REPORT"
    echo "═══════════════════════════════════════════════════════════════"
    echo
    
    echo "📊 Test Summary:"
    echo "  Total Tests:    $TESTS_TOTAL"
    echo "  Passed:         $TESTS_PASSED"
    echo "  Failed:         $TESTS_FAILED"
    echo "  Warnings:       $TESTS_WARNINGS"
    echo
    
    local success_rate=0
    if [[ $TESTS_TOTAL -gt 0 ]]; then
        success_rate=$((TESTS_PASSED * 100 / TESTS_TOTAL))
    fi
    
    echo "  Success Rate:   $success_rate%"
    echo
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        if [[ $TESTS_WARNINGS -eq 0 ]]; then
            log_success "🎉 ALL TESTS PASSED! Agent-orchestrator connection is ready."
        else
            log_warning "✅ All critical tests passed, but $TESTS_WARNINGS warnings need attention."
        fi
        echo
        echo "🚀 Next Steps:"
        echo "1. Start the orchestrator: cd 02-Orchestrator && python main.py"
        echo "2. Start the agents: sudo systemctl start stavily-*-agent"
        echo "3. Monitor agent logs: sudo journalctl -u stavily-*-agent -f"
        echo "4. Check agent status in orchestrator dashboard"
    else
        log_error "❌ $TESTS_FAILED test(s) failed. Connection setup needs attention."
        echo
        echo "🔧 Required Actions:"
        echo "1. Review failed tests above"
        echo "2. Check orchestrator is running at localhost:8000"
        echo "3. Verify agent configurations and certificates"
        echo "4. Re-run tests after fixes: ./scripts/test-agent-connection.sh"
    fi
    
    echo
    echo "📋 Test Categories:"
    echo "  • Prerequisites and tools"
    echo "  • Orchestrator health and API"
    echo "  • Agent registration and deregistration"
    echo "  • Agent health and metrics endpoints"
    echo "  • Configuration files and JWT tokens"
    echo
    echo "📖 Documentation:"
    echo "  • Setup Guide: 01-Agents/configs/README.md"
    echo "  • Troubleshooting: Run './scripts/validate-security.sh' for detailed checks"
    echo
    
    return $TESTS_FAILED
}

# Main test function
main() {
    echo "═══════════════════════════════════════════════════════════════"
    echo "           STAVILY AGENT-ORCHESTRATOR CONNECTION TEST"
    echo "═══════════════════════════════════════════════════════════════"
    echo
    log_info "Testing agent connection to orchestrator at $ORCHESTRATOR_URL"
    log_info "Hostname: $HOSTNAME"
    log_info "Timeout: ${TIMEOUT}s"
    echo
    
    # Run all tests
    check_prerequisites
    test_orchestrator_health
    test_agent_registration "sensor"
    test_agent_registration "action"
    test_agent_health_endpoints
    test_agent_metrics_endpoints
    test_agent_configurations
    test_jwt_tokens
    test_orchestrator_api_endpoints
    
    # Generate final report
    generate_test_report
    
    return $?
}

# Script options
case "${1:-test}" in
    "test")
        main
        ;;
    "registration")
        check_prerequisites
        test_orchestrator_health
        test_agent_registration "sensor"
        test_agent_registration "action"
        generate_test_report
        ;;
    "health")
        test_agent_health_endpoints
        test_agent_metrics_endpoints
        generate_test_report
        ;;
    "config")
        test_agent_configurations
        test_jwt_tokens
        generate_test_report
        ;;
    "orchestrator")
        test_orchestrator_health
        test_orchestrator_api_endpoints
        generate_test_report
        ;;
    *)
        echo "Usage: $0 [test|registration|health|config|orchestrator]"
        echo "  test:          Run complete connection test (default)"
        echo "  registration:  Test agent registration only"
        echo "  health:        Test agent health endpoints only"
        echo "  config:        Test configurations and tokens only"
        echo "  orchestrator:  Test orchestrator endpoints only"
        echo
        echo "Environment variables:"
        echo "  TEST_TIMEOUT:  Request timeout in seconds (default: 30)"
        exit 1
        ;;
esac 