# Docker Deployment

Deploy Pentora using Docker containers for portable, reproducible, and scalable deployments.

## Overview

Docker deployment is ideal for:
- Consistent environments across dev/staging/production
- Easy version management and rollback
- Container orchestration (Kubernetes, Docker Swarm)
- CI/CD pipeline integration
- Cloud-native deployments
- Development and testing

## Quick Start

### Pull and Run

```bash
# Pull official image
docker pull pentora/pentora:latest

# Run simple scan
docker run --rm pentora/pentora:latest scan 192.168.1.0/24

# Run with output
docker run --rm -v $(pwd):/output pentora/pentora:latest scan 192.168.1.100 -o /output/results.json
```

### Interactive Mode

```bash
# Start interactive shell
docker run -it --rm pentora/pentora:latest /bin/bash

# Inside container
pentora scan 192.168.1.100
pentora workspace list
```

## Docker Images

### Official Images

Available on Docker Hub: `pentora/pentora`

Tags:
- `latest` - Latest stable release
- `1.0.0` - Specific version
- `1.0` - Minor version (auto-updates patches)
- `1` - Major version (auto-updates minor/patches)
- `edge` - Development builds (unstable)
- `alpine` - Alpine-based lightweight image
- `enterprise` - Enterprise Edition (requires license)

```bash
# Pull specific version
docker pull pentora/pentora:1.0.0

# Pull Alpine variant
docker pull pentora/pentora:alpine

# Pull Enterprise Edition
docker pull pentora/pentora:enterprise
```

### Image Variants

#### Standard Image (Debian-based)

```dockerfile
FROM pentora/pentora:latest
# Size: ~150 MB
# Base: debian:bookworm-slim
# Includes: Full toolchain, debug tools
```

#### Alpine Image (Lightweight)

```dockerfile
FROM pentora/pentora:alpine
# Size: ~50 MB
# Base: alpine:3.18
# Includes: Minimal runtime only
```

#### Enterprise Image

```dockerfile
FROM pentora/pentora:enterprise
# Includes: Enterprise features, web UI, distributed scanning
```

## Basic Usage

### One-Time Scans

```bash
# Simple scan
docker run --rm pentora/pentora scan 192.168.1.100

# Scan with vulnerability assessment
docker run --rm pentora/pentora scan 192.168.1.100 --vuln

# Scan network range
docker run --rm pentora/pentora scan 192.168.1.0/24 --profile quick

# Discovery only
docker run --rm pentora/pentora scan 10.0.0.0/16 --only-discover
```

### Persistent Workspace

```bash
# Create workspace directory
mkdir -p ~/pentora-workspace

# Run with workspace persistence
docker run --rm \
  -v ~/pentora-workspace:/workspace \
  -e PENTORA_WORKSPACE_DIR=/workspace \
  pentora/pentora scan 192.168.1.0/24

# List scans
docker run --rm \
  -v ~/pentora-workspace:/workspace \
  -e PENTORA_WORKSPACE_DIR=/workspace \
  pentora/pentora workspace list

# View specific scan
docker run --rm \
  -v ~/pentora-workspace:/workspace \
  -e PENTORA_WORKSPACE_DIR=/workspace \
  pentora/pentora workspace show <scan-id>
```

### Configuration Files

```bash
# Create config directory
mkdir -p ~/pentora-config

# Create config file
cat > ~/pentora-config/config.yaml <<EOF
scanner:
  default_profile: standard
  rate: 2000
  concurrency: 200

logging:
  level: debug
  format: json
EOF

# Run with custom config
docker run --rm \
  -v ~/pentora-config:/config \
  -e PENTORA_CONFIG=/config/config.yaml \
  pentora/pentora scan 192.168.1.0/24
```

## Docker Compose

### Standalone Scanner

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pentora:
    image: pentora/pentora:latest
    container_name: pentora
    volumes:
      - ./workspace:/workspace
      - ./config:/config
    environment:
      - PENTORA_WORKSPACE_DIR=/workspace
      - PENTORA_CONFIG=/config/config.yaml
      - PENTORA_LOG_LEVEL=info
    network_mode: host
    cap_add:
      - NET_RAW
      - NET_ADMIN
    command: ["scan", "192.168.1.0/24", "--profile", "standard"]
```

Run:

```bash
docker-compose up
```

### Server Mode

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pentora-server:
    image: pentora/pentora:latest
    container_name: pentora-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./workspace:/workspace
      - ./config:/config
      - ./logs:/logs
    environment:
      - PENTORA_WORKSPACE_DIR=/workspace
      - PENTORA_CONFIG=/config/config.yaml
      - PENTORA_LOG_LEVEL=info
    networks:
      - pentora-network
    cap_add:
      - NET_RAW
      - NET_ADMIN
    command: ["server", "start", "--bind", "0.0.0.0:8080"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

networks:
  pentora-network:
    driver: bridge
```

Create configuration file `config/config.yaml`:

```yaml
server:
  bind: 0.0.0.0:8080
  workers: 4
  api:
    enabled: true
    auth: true
  queue:
    max_jobs: 1000

workspace:
  dir: /workspace
  enabled: true

logging:
  level: info
  format: json
  file:
    enabled: true
    path: /logs/pentora.log
```

Start services:

```bash
# Start in background
docker-compose up -d

# View logs
docker-compose logs -f pentora-server

# Check status
docker-compose ps

# Stop services
docker-compose down
```

### Complete Stack with Database (Enterprise)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: pentora-db
    restart: unless-stopped
    environment:
      - POSTGRES_DB=pentora
      - POSTGRES_USER=pentora
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - pentora-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U pentora"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: pentora-redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    networks:
      - pentora-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  pentora:
    image: pentora/pentora:enterprise
    container_name: pentora-server
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    ports:
      - "8080:8080"
      - "443:443"
    volumes:
      - ./workspace:/workspace
      - ./config:/config
      - ./logs:/logs
      - ./tls:/tls
    environment:
      - PENTORA_WORKSPACE_DIR=/workspace
      - PENTORA_CONFIG=/config/config.yaml
      - PENTORA_DB_HOST=postgres
      - PENTORA_DB_PORT=5432
      - PENTORA_DB_NAME=pentora
      - PENTORA_DB_USER=pentora
      - PENTORA_DB_PASSWORD=${DB_PASSWORD}
      - PENTORA_REDIS_HOST=redis
      - PENTORA_REDIS_PORT=6379
      - PENTORA_LICENSE_FILE=/config/license.key
    networks:
      - pentora-network
    cap_add:
      - NET_RAW
      - NET_ADMIN
    command: ["server", "start"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  nginx:
    image: nginx:alpine
    container_name: pentora-nginx
    restart: unless-stopped
    depends_on:
      - pentora
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./tls:/etc/nginx/tls:ro
    networks:
      - pentora-network

volumes:
  postgres-data:
  redis-data:

networks:
  pentora-network:
    driver: bridge
```

Create `.env` file:

```bash
DB_PASSWORD=changeme
```

Create `nginx/nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream pentora {
        server pentora:8080;
    }

    server {
        listen 80;
        server_name pentora.company.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name pentora.company.com;

        ssl_certificate /etc/nginx/tls/cert.pem;
        ssl_certificate_key /etc/nginx/tls/key.pem;

        location / {
            proxy_pass http://pentora;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /health {
            proxy_pass http://pentora;
            access_log off;
        }
    }
}
```

Start stack:

```bash
docker-compose up -d
```

## Kubernetes Deployment

### Simple Deployment

Create `pentora-deployment.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: pentora

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: pentora-config
  namespace: pentora
data:
  config.yaml: |
    server:
      bind: 0.0.0.0:8080
      workers: 4
    workspace:
      dir: /workspace
      enabled: true
    logging:
      level: info
      format: json

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pentora-workspace
  namespace: pentora
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pentora-server
  namespace: pentora
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pentora
  template:
    metadata:
      labels:
        app: pentora
    spec:
      containers:
      - name: pentora
        image: pentora/pentora:latest
        imagePullPolicy: Always
        command: ["pentora", "server", "start"]
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: PENTORA_WORKSPACE_DIR
          value: /workspace
        - name: PENTORA_CONFIG
          value: /config/config.yaml
        volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: config
          mountPath: /config
        resources:
          requests:
            cpu: 500m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 8Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        securityContext:
          capabilities:
            add:
            - NET_RAW
            - NET_ADMIN
      volumes:
      - name: workspace
        persistentVolumeClaim:
          claimName: pentora-workspace
      - name: config
        configMap:
          name: pentora-config

---
apiVersion: v1
kind: Service
metadata:
  name: pentora-service
  namespace: pentora
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: pentora
```

Deploy:

```bash
kubectl apply -f pentora-deployment.yaml

# Check status
kubectl -n pentora get pods
kubectl -n pentora get svc

# View logs
kubectl -n pentora logs -f deployment/pentora-server

# Access service
kubectl -n pentora port-forward service/pentora-service 8080:80
```

### StatefulSet for HA

Create `pentora-statefulset.yaml`:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: pentora
  namespace: pentora
spec:
  serviceName: pentora
  replicas: 3
  selector:
    matchLabels:
      app: pentora
  template:
    metadata:
      labels:
        app: pentora
    spec:
      containers:
      - name: pentora
        image: pentora/pentora:enterprise
        command: ["pentora", "server", "start"]
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: PENTORA_WORKSPACE_DIR
          value: /workspace
        - name: PENTORA_REDIS_HOST
          value: redis-service
        volumeMounts:
        - name: workspace
          mountPath: /workspace
        resources:
          requests:
            cpu: 1000m
            memory: 4Gi
          limits:
            cpu: 4000m
            memory: 16Gi
  volumeClaimTemplates:
  - metadata:
      name: workspace
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

## Building Custom Images

### Dockerfile

Create custom `Dockerfile`:

```dockerfile
FROM pentora/pentora:latest

# Install additional tools
RUN apt-get update && apt-get install -y \
    nmap \
    masscan \
    jq \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy custom configuration
COPY config/config.yaml /etc/pentora/config.yaml

# Copy custom fingerprints
COPY fingerprints/ /usr/share/pentora/fingerprints/

# Copy custom scripts
COPY scripts/ /usr/local/bin/

# Set default environment variables
ENV PENTORA_CONFIG=/etc/pentora/config.yaml
ENV PENTORA_WORKSPACE_DIR=/workspace

# Expose ports
EXPOSE 8080

# Default command
CMD ["pentora", "server", "start"]
```

Build and push:

```bash
# Build image
docker build -t mycompany/pentora:custom .

# Test locally
docker run --rm mycompany/pentora:custom version

# Push to registry
docker push mycompany/pentora:custom
```

### Multi-Stage Build

Create optimized `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy source
COPY . .

# Build binary
RUN go build -o pentora -ldflags="-s -w" ./cmd/pentora

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libcap \
    && rm -rf /var/cache/apk/*

# Copy binary from builder
COPY --from=builder /build/pentora /usr/local/bin/pentora

# Set capabilities
RUN setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/pentora

# Create non-root user
RUN adduser -D -u 1000 pentora

# Create directories
RUN mkdir -p /workspace /config && \
    chown -R pentora:pentora /workspace /config

USER pentora

WORKDIR /workspace

EXPOSE 8080

CMD ["pentora", "server", "start"]
```

Build:

```bash
docker build -t mycompany/pentora:optimized .
```

## Security Considerations

### Network Mode

```bash
# Host network (full access)
docker run --network host pentora/pentora scan 192.168.1.0/24

# Bridge network (isolated)
docker run --network bridge pentora/pentora scan scanme.nmap.org

# Custom network
docker network create pentora-net
docker run --network pentora-net pentora/pentora scan targets
```

### Capabilities

```bash
# Add required capabilities
docker run --cap-add NET_RAW --cap-add NET_ADMIN pentora/pentora scan 192.168.1.0/24

# Drop unnecessary capabilities
docker run --cap-drop ALL --cap-add NET_RAW pentora/pentora scan 192.168.1.0/24

# Run as non-root (if capabilities set in image)
docker run --user pentora pentora/pentora scan 192.168.1.0/24
```

### Read-Only Filesystem

```bash
docker run --read-only \
  -v /tmp --tmpfs /tmp \
  -v $(pwd)/workspace:/workspace \
  pentora/pentora scan 192.168.1.0/24
```

### Security Scanning

```bash
# Scan image for vulnerabilities
docker scan pentora/pentora:latest

# Use Trivy
trivy image pentora/pentora:latest

# Use Snyk
snyk container test pentora/pentora:latest
```

## Performance Optimization

### Resource Limits

```bash
# Set CPU and memory limits
docker run --cpus=2 --memory=4g pentora/pentora scan 192.168.1.0/24

# CPU shares
docker run --cpu-shares=512 pentora/pentora scan 192.168.1.0/24

# Memory reservation
docker run --memory=4g --memory-reservation=2g pentora/pentora scan 192.168.1.0/24
```

In Docker Compose:

```yaml
services:
  pentora:
    image: pentora/pentora
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '0.5'
          memory: 1G
```

### Volume Performance

```bash
# Use tmpfs for temporary data
docker run --tmpfs /tmp:rw,size=1g pentora/pentora scan 192.168.1.0/24

# Use volumes instead of bind mounts
docker volume create pentora-workspace
docker run -v pentora-workspace:/workspace pentora/pentora scan 192.168.1.0/24
```

## Monitoring and Logging

### Container Logs

```bash
# View logs
docker logs pentora-server

# Follow logs
docker logs -f pentora-server

# Last 100 lines
docker logs --tail 100 pentora-server

# Since timestamp
docker logs --since 2024-10-06T10:00:00 pentora-server
```

### Log Drivers

```bash
# JSON file (default)
docker run --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  pentora/pentora server start

# Syslog
docker run --log-driver syslog \
  --log-opt syslog-address=udp://logs.company.com:514 \
  pentora/pentora server start

# Fluentd
docker run --log-driver fluentd \
  --log-opt fluentd-address=localhost:24224 \
  pentora/pentora server start
```

### Health Checks

```bash
# Add health check
docker run --health-cmd="curl -f http://localhost:8080/health || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  pentora/pentora server start

# Check health status
docker inspect --format='{{.State.Health.Status}}' pentora-server
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Security Scan with Pentora

on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - name: Run Pentora Scan
        run: |
          docker run --rm \
            -v $(pwd):/output \
            pentora/pentora:latest \
            scan ${{ secrets.SCAN_TARGETS }} \
            -o /output/results.json

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: scan-results
          path: results.json
```

### GitLab CI

```yaml
pentora_scan:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker pull pentora/pentora:latest
    - docker run --rm pentora/pentora scan $SCAN_TARGETS -o results.json
  artifacts:
    paths:
      - results.json
```

## Troubleshooting

### Permission Denied

```bash
# Add capabilities
docker run --cap-add NET_RAW --cap-add NET_ADMIN pentora/pentora scan 192.168.1.0/24

# Run as root
docker run --user root pentora/pentora scan 192.168.1.0/24
```

### Network Not Reachable

```bash
# Use host network
docker run --network host pentora/pentora scan 192.168.1.0/24

# Check network connectivity
docker run --rm pentora/pentora ping 192.168.1.1
```

### Container Crashes

```bash
# Check logs
docker logs pentora-server

# Inspect container
docker inspect pentora-server

# Check resource limits
docker stats pentora-server
```

## Next Steps

- [Standalone Deployment](/docs/deployment/standalone) - CLI deployment
- [Server Mode Deployment](/docs/deployment/server-mode) - Systemd service
- [Air-Gapped Deployment](/docs/deployment/air-gapped) - Offline deployment
- [Enterprise Features](/docs/enterprise/overview) - Scale and features
