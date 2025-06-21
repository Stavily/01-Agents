# Stavily Agents Deployment Guide

This guide provides detailed instructions for deploying Stavily agents in production environments with enterprise-grade security, monitoring, and reliability.

## Table of Contents

1. [Production Deployment Architecture](#production-deployment-architecture)
2. [Security Hardening](#security-hardening)
3. [High Availability Setup](#high-availability-setup)
4. [Monitoring and Observability](#monitoring-and-observability)
5. [Backup and Recovery](#backup-and-recovery)
6. [Troubleshooting](#troubleshooting)

## Production Deployment Architecture

### Recommended Infrastructure

```
Production Environment
├── Load Balancer (HAProxy/NGINX)
├── Agent Cluster (3+ nodes)
│   ├── Sensor Agents (monitoring tier)
│   ├── Action Agents (execution tier)
│   └── Shared Storage (NFS/GlusterFS)
├── Monitoring Stack
│   ├── Prometheus
│   ├── Grafana
│   └── AlertManager
└── Log Aggregation
    ├── Elasticsearch
    ├── Logstash
    └── Kibana
```

### Base Directory Structure for Production

```bash
# Production base directory structure
/opt/stavily/
├── agents/
│   ├── sensor/
│   │   ├── config/
│   │   │   ├── agent.yaml
│   │   │   ├── plugins/
│   │   │   └── certificates/
│   │   ├── data/
│   │   │   ├── plugins/
│   │   │   ├── cache/
│   │   │   └── state/
│   │   ├── logs/
│   │   │   ├── agent.log
│   │   │   ├── plugins/
│   │   │   └── audit/
│   │   └── tmp/
│   └── action/
│       ├── config/
│       ├── data/
│       ├── logs/
│       └── tmp/
├── shared/
│   ├── certificates/
│   ├── plugins/
│   └── backups/
└── monitoring/
    ├── prometheus/
    ├── grafana/
    └── logs/
```

## Security Hardening

### 1. Certificate Management

#### Production Certificate Setup

```bash
#!/bin/bash
# Production certificate deployment script

CERT_DIR="/opt/stavily/shared/certificates"
AGENT_USER="stavily"

# Create secure certificate directory
sudo mkdir -p $CERT_DIR/{ca,client,server,backup}
sudo chown -R $AGENT_USER:$AGENT_USER $CERT_DIR
sudo chmod 700 $CERT_DIR

# Generate strong certificates (example with OpenSSL)
# Note: In production, use your PKI infrastructure

# Generate CA private key
sudo openssl genrsa -aes256 -out $CERT_DIR/ca/ca-key.pem 4096

# Generate CA certificate
sudo openssl req -new -x509 -days 365 -key $CERT_DIR/ca/ca-key.pem \
    -sha256 -out $CERT_DIR/ca/ca.pem \
    -subj "/C=US/ST=CA/L=San Francisco/O=YourOrg/CN=Stavily CA"

# Generate client private key
sudo openssl genrsa -out $CERT_DIR/client/client-key.pem 4096

# Generate client certificate signing request
sudo openssl req -subj "/CN=stavily-agent" -new \
    -key $CERT_DIR/client/client-key.pem \
    -out $CERT_DIR/client/client.csr

# Sign client certificate
sudo openssl x509 -req -days 365 -in $CERT_DIR/client/client.csr \
    -CA $CERT_DIR/ca/ca.pem -CAkey $CERT_DIR/ca/ca-key.pem \
    -out $CERT_DIR/client/client.pem -sha256

# Set proper permissions
sudo chmod 400 $CERT_DIR/ca/ca-key.pem
sudo chmod 444 $CERT_DIR/ca/ca.pem
sudo chmod 400 $CERT_DIR/client/client-key.pem
sudo chmod 444 $CERT_DIR/client/client.pem
sudo chown -R $AGENT_USER:$AGENT_USER $CERT_DIR
```

#### Certificate Rotation Script

```bash
#!/bin/bash
# Certificate rotation script for production

CERT_DIR="/opt/stavily/shared/certificates"
BACKUP_DIR="/opt/stavily/shared/backups/certificates"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup
mkdir -p $BACKUP_DIR/$DATE
cp -r $CERT_DIR/* $BACKUP_DIR/$DATE/

# Download new certificates from Stavily
curl -H "Authorization: Bearer $STAVILY_API_TOKEN" \
    "https://api.stavily.com/v1/agents/certificates/bundle" \
    -o /tmp/certs_$DATE.tar.gz

# Extract and install new certificates
cd /tmp
tar -xzf certs_$DATE.tar.gz
sudo cp client.crt $CERT_DIR/client/client.pem
sudo cp client.key $CERT_DIR/client/client-key.pem
sudo cp ca.crt $CERT_DIR/ca/ca.pem

# Set permissions
sudo chown -R stavily:stavily $CERT_DIR
sudo chmod 400 $CERT_DIR/client/client-key.pem
sudo chmod 444 $CERT_DIR/client/client.pem
sudo chmod 444 $CERT_DIR/ca/ca.pem

# Restart agents to pick up new certificates
sudo systemctl restart sensor-agent-{AGENT_ID}
sudo systemctl restart action-agent-{AGENT_ID}

# Cleanup
rm -f /tmp/certs_$DATE.tar.gz /tmp/client.* /tmp/ca.crt

echo "Certificate rotation completed at $(date)"
```

### 2. Network Security

#### Firewall Configuration (iptables)

```bash
#!/bin/bash
# Production firewall rules

# Flush existing rules
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X

# Default policies
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Allow loopback
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow SSH (change port as needed)
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Allow health checks
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 8081 -s 10.0.0.0/8 -j ACCEPT

# Allow metrics collection
iptables -A INPUT -p tcp --dport 9090 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 9091 -s 10.0.0.0/8 -j ACCEPT

# Allow outbound HTTPS to Stavily
iptables -A OUTPUT -p tcp --dport 443 -d agents.stavily.com -j ACCEPT

# Log dropped packets
iptables -A INPUT -j LOG --log-prefix "DROPPED: "

# Save rules
iptables-save > /etc/iptables/rules.v4
```

### 3. System Hardening

#### SELinux/AppArmor Configuration

```bash
# SELinux policy for Stavily agents
# /etc/selinux/local/stavily-agent.te

module stavily-agent 1.0;

require {
    type unconfined_t;
    type bin_t;
    type etc_t;
    type var_log_t;
    class file { read write execute };
    class dir { read write };
}

# Allow agent to read configuration
allow unconfined_t etc_t:file { read };

# Allow agent to write logs
allow unconfined_t var_log_t:file { write };

# Allow agent execution
allow unconfined_t bin_t:file { execute };
```

## High Availability Setup

### 1. Multi-Node Deployment

#### Docker Swarm Configuration

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  sensor-agent:
    image: stavily/sensor-agent:latest
    deploy:
      replicas: 3
      placement:
        constraints:
          - node.role == worker
        preferences:
          - spread: node.labels.zone
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    networks:
      - stavily-network
    volumes:
      - stavily-config:/app/agent-{AGENT_ID}/config:ro
      - stavily-data:/app/agent-{AGENT_ID}/data
      - stavily-logs:/app/agent-{AGENT_ID}/logs
      - /var/run/docker.sock:/var/run/docker.sock
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  action-agent:
    image: stavily/action-agent:latest
    deploy:
      replicas: 2
      placement:
        constraints:
          - node.role == worker
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '1.0'
          memory: 1G
    networks:
      - stavily-network
    volumes:
      - stavily-config:/app/agent-{AGENT_ID}/config:ro
      - stavily-data:/app/agent-{AGENT_ID}/data
      - stavily-logs:/app/agent-{AGENT_ID}/logs
      - /var/run/docker.sock:/var/run/docker.sock

networks:
  stavily-network:
    driver: overlay
    encrypted: true

volumes:
  stavily-config:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs-server,rw
      device: ":/opt/stavily/config"
  stavily-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs-server,rw
      device: ":/opt/stavily/data"
  stavily-logs:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs-server,rw
      device: ":/opt/stavily/logs"
```

#### Kubernetes HA Deployment

```yaml
# k8s-ha-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sensor-agent
  namespace: stavily-agents
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: sensor-agent
  template:
    metadata:
      labels:
        app: sensor-agent
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - sensor-agent
              topologyKey: kubernetes.io/hostname
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
      - name: sensor-agent
        image: stavily/sensor-agent:latest
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
        volumeMounts:
        - name: config
          mountPath: /app/agent-{AGENT_ID}/config
          readOnly: true
        - name: data
          mountPath: /app/agent-{AGENT_ID}/data
        - name: logs
          mountPath: /app/agent-{AGENT_ID}/logs
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
      volumes:
      - name: config
        configMap:
          name: agent-config
      - name: data
        persistentVolumeClaim:
          claimName: agent-data
      - name: logs
        persistentVolumeClaim:
          claimName: agent-logs

---
apiVersion: v1
kind: Service
metadata:
  name: sensor-agent-service
  namespace: stavily-agents
spec:
  selector:
    app: sensor-agent
  ports:
  - name: health
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP

---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: sensor-agent-pdb
  namespace: stavily-agents
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: sensor-agent
```

## Monitoring and Observability

### 1. Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "stavily-alerts.yml"

scrape_configs:
  - job_name: 'sensor-agent-{AGENT_ID}-agents'
    static_configs:
      - targets: ['sensor-agent-1:9090', 'sensor-agent-2:9090', 'sensor-agent-3:9090']
    scrape_interval: 30s
    metrics_path: /metrics

  - job_name: 'action-agent-{AGENT_ID}-agents'
    static_configs:
      - targets: ['action-agent-1:9091', 'action-agent-2:9091']
    scrape_interval: 30s
    metrics_path: /metrics

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### 2. Alert Rules

```yaml
# stavily-alerts.yml
groups:
- name: stavily-agents
  rules:
  - alert: AgentDown
    expr: up{job=~"stavily-.*-agents"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Stavily agent is down"
      description: "Agent {{ $labels.instance }} has been down for more than 1 minute."

  - alert: HighMemoryUsage
    expr: (stavily_agent_memory_usage_bytes / stavily_agent_memory_limit_bytes) * 100 > 80
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage on Stavily agent"
      description: "Agent {{ $labels.instance }} memory usage is above 80%"

  - alert: PluginFailures
    expr: increase(stavily_agent_plugin_failures_total[5m]) > 5
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High plugin failure rate"
      description: "Agent {{ $labels.instance }} has had {{ $value }} plugin failures in the last 5 minutes"

  - alert: CertificateExpiry
    expr: (stavily_agent_certificate_expiry_timestamp - time()) / 86400 < 30
    for: 1h
    labels:
      severity: warning
    annotations:
      summary: "Certificate expiring soon"
      description: "Agent {{ $labels.instance }} certificate expires in {{ $value }} days"

  - alert: APIConnectionFailures
    expr: increase(stavily_agent_api_connection_failures_total[10m]) > 10
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Frequent API connection failures"
      description: "Agent {{ $labels.instance }} has had {{ $value }} API connection failures in the last 10 minutes"
```

### 3. Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Stavily Agents Overview",
    "panels": [
      {
        "title": "Agent Status",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(up{job=~\"stavily-.*-agents\"})",
            "legendFormat": "Active Agents"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "stavily_agent_memory_usage_bytes",
            "legendFormat": "{{ instance }}"
          }
        ]
      },
      {
        "title": "Plugin Execution Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(stavily_agent_plugin_executions_total[5m])",
            "legendFormat": "{{ instance }} - {{ plugin }}"
          }
        ]
      },
      {
        "title": "API Response Times",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(stavily_agent_api_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

## Backup and Recovery

### 1. Backup Script

```bash
#!/bin/bash
# Production backup script for Stavily agents

BACKUP_DIR="/opt/stavily/shared/backups"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR/$DATE

# Backup configuration
tar -czf $BACKUP_DIR/$DATE/config_$DATE.tar.gz \
    /opt/stavily/agents/*/config/

# Backup agent data (excluding logs and tmp)
tar -czf $BACKUP_DIR/$DATE/data_$DATE.tar.gz \
    --exclude='*/logs/*' \
    --exclude='*/tmp/*' \
    /opt/stavily/agents/*/data/

# Backup certificates
tar -czf $BACKUP_DIR/$DATE/certificates_$DATE.tar.gz \
    /opt/stavily/shared/certificates/

# Create manifest
cat > $BACKUP_DIR/$DATE/manifest.txt << EOF
Backup Date: $DATE
Hostname: $(hostname)
Agent Version: $(sensor-agent --version)
Config Files: $(find /opt/stavily/agents/*/config/ -name "*.yaml" | wc -l)
Plugin Count: $(find /opt/stavily/agents/*/data/plugins/ -name "*.so" | wc -l)
Certificate Expiry: $(openssl x509 -in /opt/stavily/shared/certificates/client/client.pem -noout -enddate)
EOF

# Sync to remote storage (example with rsync)
rsync -av $BACKUP_DIR/$DATE/ backup-server:/backups/stavily/$DATE/

# Cleanup old backups
find $BACKUP_DIR -type d -mtime +$RETENTION_DAYS -exec rm -rf {} \;

echo "Backup completed: $BACKUP_DIR/$DATE"
```

### 2. Recovery Procedures

```bash
#!/bin/bash
# Recovery script for Stavily agents

BACKUP_DATE="$1"
BACKUP_DIR="/opt/stavily/shared/backups"

if [ -z "$BACKUP_DATE" ]; then
    echo "Usage: $0 <backup_date>"
    echo "Available backups:"
    ls -la $BACKUP_DIR/
    exit 1
fi

# Stop agents
sudo systemctl stop sensor-agent-{AGENT_ID}
sudo systemctl stop action-agent-{AGENT_ID}

# Backup current state
mv /opt/stavily/agents /opt/stavily/agents.backup.$(date +%s)

# Restore from backup
cd $BACKUP_DIR/$BACKUP_DATE
tar -xzf config_$BACKUP_DATE.tar.gz -C /
tar -xzf data_$BACKUP_DATE.tar.gz -C /
tar -xzf certificates_$BACKUP_DATE.tar.gz -C /

# Set proper permissions
chown -R stavily:stavily /opt/stavily/agents
chown -R stavily:stavily /opt/stavily/shared/certificates
chmod -R 700 /opt/stavily/shared/certificates

# Start agents
sudo systemctl start sensor-agent-{AGENT_ID}
sudo systemctl start action-agent-{AGENT_ID}

# Verify recovery
sleep 10
curl -f http://localhost:8080/health && echo "Sensor agent recovered successfully"
curl -f http://localhost:8081/health && echo "Action agent recovered successfully"
```

## Troubleshooting

### 1. Common Production Issues

#### Agent Registration Problems

```bash
# Check agent logs
sudo journalctl -u sensor-agent-{AGENT_ID} -n 100

# Verify network connectivity
curl -v https://agents.stavily.com/health

# Check certificate validity
openssl x509 -in /opt/stavily/shared/certificates/client/client.pem -noout -dates

# Test certificate authentication
curl --cert /opt/stavily/shared/certificates/client/client.pem \
     --key /opt/stavily/shared/certificates/client/client-key.pem \
     --cacert /opt/stavily/shared/certificates/ca/ca.pem \
     https://agents.stavily.com/v1/agents/register
```

#### Plugin Issues

```bash
# List loaded plugins
curl http://localhost:8080/debug/plugins

# Check plugin logs
tail -f /opt/stavily/agents/sensor/logs/plugins/*.log

# Verify plugin permissions
ls -la /opt/stavily/agents/sensor/data/plugins/

# Test plugin manually
/opt/stavily/agents/sensor/data/plugins/test-plugin --config=/opt/stavily/agents/sensor/config/plugins/test-plugin.yaml
```

#### Performance Issues

```bash
# Monitor resource usage
top -p $(pgrep -f stavily)

# Check disk I/O
iotop -p $(pgrep -f stavily)

# Analyze memory usage
pmap -x $(pgrep -f sensor-agent-{AGENT_ID})

# Review metrics
curl http://localhost:9090/metrics | grep stavily_agent
```

### 2. Emergency Procedures

#### Emergency Stop

```bash
#!/bin/bash
# Emergency stop script

echo "Initiating emergency stop of Stavily agents..."

# Stop all agents immediately
sudo pkill -TERM -f stavily
sleep 5
sudo pkill -KILL -f stavily

# Stop services
sudo systemctl stop sensor-agent-{AGENT_ID}
sudo systemctl stop action-agent-{AGENT_ID}

# Disable services to prevent restart
sudo systemctl disable sensor-agent-{AGENT_ID}
sudo systemctl disable action-agent-{AGENT_ID}

echo "Emergency stop completed. All Stavily agents stopped."
```

#### Quick Recovery

```bash
#!/bin/bash
# Quick recovery script

echo "Starting quick recovery..."

# Clear any corrupted state
rm -rf /opt/stavily/agents/*/tmp/*
rm -rf /opt/stavily/agents/*/data/cache/*

# Reset to known good configuration
cp /opt/stavily/shared/backups/config/agent.yaml.backup \
   /opt/stavily/agents/sensor/config/agent.yaml

# Start with minimal configuration
export STAVILY_LOGGING_LEVEL=debug
export STAVILY_PLUGINS_AUTO_UPDATE=false

# Start agents
sudo systemctl enable sensor-agent-{AGENT_ID}
sudo systemctl start sensor-agent-{AGENT_ID}

# Wait and verify
sleep 10
if curl -f http://localhost:8080/health; then
    echo "Sensor agent recovered successfully"
    sudo systemctl enable action-agent-{AGENT_ID}
    sudo systemctl start action-agent-{AGENT_ID}
else
    echo "Recovery failed. Check logs: sudo journalctl -u sensor-agent-{AGENT_ID} -f"
fi
```

## Production Checklist

### Pre-Deployment

- [ ] Security hardening completed
- [ ] Certificates installed and validated
- [ ] Firewall rules configured
- [ ] Monitoring stack deployed
- [ ] Backup procedures tested
- [ ] Recovery procedures documented
- [ ] Load testing completed
- [ ] Security audit passed

### Post-Deployment

- [ ] Agent registration verified
- [ ] Plugin functionality tested
- [ ] Monitoring alerts configured
- [ ] Log aggregation working
- [ ] Backup schedule active
- [ ] Performance baselines established
- [ ] Documentation updated
- [ ] Team training completed

### Ongoing Maintenance

- [ ] Certificate rotation scheduled
- [ ] Security updates applied
- [ ] Performance monitoring active
- [ ] Backup verification regular
- [ ] Capacity planning updated
- [ ] Incident response tested
- [ ] Documentation maintained
- [ ] Team knowledge updated

## Support Contacts

- **Emergency Support**: emergency@stavily.com
- **Production Issues**: production@stavily.com
- **Security Issues**: security@stavily.com
- **Documentation**: docs@stavily.com 