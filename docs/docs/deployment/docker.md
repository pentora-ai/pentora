# Docker Deployment

Deploy Vulntor using Docker containers for portable, reproducible, and scalable deployments.

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
docker pull vulntor/vulntor:latest

# Run simple scan
docker run --rm vulntor/vulntor:latest scan 192.168.1.0/24

# Run with output
docker run --rm -v $(pwd):/output vulntor/vulntor:latest scan 192.168.1.100 -o /output/results.json
```

### Interactive Mode

```bash
# Start interactive shell
docker run -it --rm vulntor/vulntor:latest /bin/bash

# Inside container
vulntor scan 192.168.1.100
vulntor storage list
```

## Docker Images

### Official Images

Available on Docker Hub: `vulntor/vulntor`

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
docker pull vulntor/vulntor:1.0.0

# Pull Alpine variant
docker pull vulntor/vulntor:alpine

# Pull Enterprise Edition
docker pull vulntor/vulntor:enterprise
```

### Image Variants

#### Standard Image (Debian-based)

```dockerfile
FROM vulntor/vulntor:latest
# Size: ~150 MB
# Base: debian:bookworm-slim
# Includes: Full toolchain, debug tools
```

#### Alpine Image (Lightweight)

```dockerfile
FROM vulntor/vulntor:alpine
# Size: ~50 MB
# Base: alpine:3.18
# Includes: Minimal runtime only
```

#### Enterprise Image

```dockerfile
FROM vulntor/vulntor:enterprise
# Includes: Enterprise features, web UI, distributed scanning
```

## Basic Usage

### One-Time Scans

```bash
# Simple scan
docker run --rm vulntor/vulntor scan 192.168.1.100

# Scan with vulnerability assessment
docker run --rm vulntor/vulntor scan 192.168.1.100 --vuln

# Scan network range
docker run --rm vulntor/vulntor scan 192.168.1.0/24 --profile quick

# Discovery only
docker run --rm vulntor/vulntor scan 10.0.0.0/16 --only-discover
```

### Persistent Storage

```bash
# Create storage directory
mkdir -p ~/vulntor-storage

# Run with storage persistence
docker run --rm \
  -v ~/vulntor-storage:/storage \
  -e VULNTOR_STORAGE_DIR=/storage \
  vulntor/vulntor scan 192.168.1.0/24

# List scans
docker run --rm \
  -v ~/vulntor-storage:/storage \
  -e VULNTOR_STORAGE_DIR=/storage \
  vulntor/vulntor storage list

# View specific scan
docker run --rm \
  -v ~/vulntor-storage:/storage \
  -e VULNTOR_STORAGE_DIR=/storage \
  vulntor/vulntor storage show <scan-id>
```

### Configuration Files

```bash
# Create config directory
mkdir -p ~/vulntor-config

# Create config file
cat > ~/vulntor-config/config.yaml <<EOF
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
  -v ~/vulntor-config:/config \
  -e VULNTOR_CONFIG=/config/config.yaml \
  vulntor/vulntor scan 192.168.1.0/24
```

## Docker Compose

### Standalone Scanner

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  vulntor:
    image: vulntor/vulntor:latest
    container_name: vulntor
    volumes:
      - ./storage:/storage
      - ./config:/config
    environment:
      - VULNTOR_STORAGE_DIR=/storage
      - VULNTOR_CONFIG=/config/config.yaml
      - VULNTOR_LOG_LEVEL=info
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
  vulntor-server:
    image: vulntor/vulntor:latest
    container_name: vulntor-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./storage:/storage
      - ./config:/config
      - ./logs:/logs
    environment:
      - VULNTOR_STORAGE_DIR=/storage
      - VULNTOR_CONFIG=/config/config.yaml
      - VULNTOR_LOG_LEVEL=info
    networks:
      - vulntor-network
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
  vulntor-network:
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

storage:
  dir: /storage
  enabled: true

logging:
  level: info
  format: json
  file:
    enabled: true
    path: /logs/vulntor.log
```

Start services:

```bash
# Start in background
docker-compose up -d

# View logs
docker-compose logs -f vulntor-server

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
    container_name: vulntor-db
    restart: unless-stopped
    environment:
      - POSTGRES_DB=vulntor
      - POSTGRES_USER=vulntor
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - vulntor-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U vulntor"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: vulntor-redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    networks:
      - vulntor-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  vulntor:
    image: vulntor/vulntor:enterprise
    container_name: vulntor-server
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
      - ./storage:/storage
      - ./config:/config
      - ./logs:/logs
      - ./tls:/tls
    environment:
      - VULNTOR_STORAGE_DIR=/storage
      - VULNTOR_CONFIG=/config/config.yaml
      - VULNTOR_DB_HOST=postgres
      - VULNTOR_DB_PORT=5432
      - VULNTOR_DB_NAME=vulntor
      - VULNTOR_DB_USER=vulntor
      - VULNTOR_DB_PASSWORD=${DB_PASSWORD}
      - VULNTOR_REDIS_HOST=redis
      - VULNTOR_REDIS_PORT=6379
      - VULNTOR_LICENSE_FILE=/config/license.key
    networks:
      - vulntor-network
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
    container_name: vulntor-nginx
    restart: unless-stopped
    depends_on:
      - vulntor
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./tls:/etc/nginx/tls:ro
    networks:
      - vulntor-network

volumes:
  postgres-data:
  redis-data:

networks:
  vulntor-network:
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
    upstream vulntor {
        server vulntor:8080;
    }

    server {
        listen 80;
        server_name vulntor.company.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name vulntor.company.com;

        ssl_certificate /etc/nginx/tls/cert.pem;
        ssl_certificate_key /etc/nginx/tls/key.pem;

        location / {
            proxy_pass http://vulntor;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location /health {
            proxy_pass http://vulntor;
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

Create `vulntor-deployment.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: vulntor

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: vulntor-config
  namespace: vulntor
data:
  config.yaml: |
    server:
      bind: 0.0.0.0:8080
      workers: 4
    storage:
      dir: /storage
      enabled: true
    logging:
      level: info
      format: json

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: vulntor-storage
  namespace: vulntor
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
  name: vulntor-server
  namespace: vulntor
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vulntor
  template:
    metadata:
      labels:
        app: vulntor
    spec:
      containers:
      - name: vulntor
        image: vulntor/vulntor:latest
        imagePullPolicy: Always
        command: ["vulntor", "server", "start"]
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: VULNTOR_STORAGE_DIR
          value: /storage
        - name: VULNTOR_CONFIG
          value: /config/config.yaml
        volumeMounts:
        - name: storage
          mountPath: /storage
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
      - name: storage
        persistentVolumeClaim:
          claimName: vulntor-storage
      - name: config
        configMap:
          name: vulntor-config

---
apiVersion: v1
kind: Service
metadata:
  name: vulntor-service
  namespace: vulntor
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: vulntor
```

Deploy:

```bash
kubectl apply -f vulntor-deployment.yaml

# Check status
kubectl -n vulntor get pods
kubectl -n vulntor get svc

# View logs
kubectl -n vulntor logs -f deployment/vulntor-server

# Access service
kubectl -n vulntor port-forward service/vulntor-service 8080:80
```

### StatefulSet for HA

Create `vulntor-statefulset.yaml`:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: vulntor
  namespace: vulntor
spec:
  serviceName: vulntor
  replicas: 3
  selector:
    matchLabels:
      app: vulntor
  template:
    metadata:
      labels:
        app: vulntor
    spec:
      containers:
      - name: vulntor
        image: vulntor/vulntor:enterprise
        command: ["vulntor", "server", "start"]
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: VULNTOR_STORAGE_DIR
          value: /storage
        - name: VULNTOR_REDIS_HOST
          value: redis-service
        volumeMounts:
        - name: storage
          mountPath: /storage
        resources:
          requests:
            cpu: 1000m
            memory: 4Gi
          limits:
            cpu: 4000m
            memory: 16Gi
  volumeClaimTemplates:
  - metadata:
      name: storage
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
FROM vulntor/vulntor:latest

# Install additional tools
RUN apt-get update && apt-get install -y \
    nmap \
    masscan \
    jq \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy custom configuration
COPY config/config.yaml /etc/vulntor/config.yaml

# Copy custom fingerprints
COPY fingerprints/ /usr/share/vulntor/fingerprints/

# Copy custom scripts
COPY scripts/ /usr/local/bin/

# Set default environment variables
ENV VULNTOR_CONFIG=/etc/vulntor/config.yaml
ENV VULNTOR_STORAGE_DIR=/storage

# Expose ports
EXPOSE 8080

# Default command
CMD ["vulntor", "server", "start"]
```

Build and push:

```bash
# Build image
docker build -t mycompany/vulntor:custom .

# Test locally
docker run --rm mycompany/vulntor:custom version

# Push to registry
docker push mycompany/vulntor:custom
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
RUN go build -o vulntor -ldflags="-s -w" ./cmd/vulntor

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    libcap \
    && rm -rf /var/cache/apk/*

# Copy binary from builder
COPY --from=builder /build/vulntor /usr/local/bin/vulntor

# Set capabilities
RUN setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/vulntor

# Create non-root user
RUN adduser -D -u 1000 vulntor

# Create directories
RUN mkdir -p /storage /config && \
    chown -R vulntor:vulntor /storage /config

USER vulntor

WORKDIR /storage

EXPOSE 8080

CMD ["vulntor", "server", "start"]
```

Build:

```bash
docker build -t mycompany/vulntor:optimized .
```

## Security Considerations

### Network Mode

```bash
# Host network (full access)
docker run --network host vulntor/vulntor scan 192.168.1.0/24

# Bridge network (isolated)
docker run --network bridge vulntor/vulntor scan scanme.nmap.org

# Custom network
docker network create vulntor-net
docker run --network vulntor-net vulntor/vulntor scan targets
```

### Capabilities

```bash
# Add required capabilities
docker run --cap-add NET_RAW --cap-add NET_ADMIN vulntor/vulntor scan 192.168.1.0/24

# Drop unnecessary capabilities
docker run --cap-drop ALL --cap-add NET_RAW vulntor/vulntor scan 192.168.1.0/24

# Run as non-root (if capabilities set in image)
docker run --user vulntor vulntor/vulntor scan 192.168.1.0/24
```

### Read-Only Filesystem

```bash
docker run --read-only \
  -v /tmp --tmpfs /tmp \
  -v $(pwd)/storage:/storage \
  vulntor/vulntor scan 192.168.1.0/24
```

### Security Scanning

```bash
# Scan image for vulnerabilities
docker scan vulntor/vulntor:latest

# Use Trivy
trivy image vulntor/vulntor:latest

# Use Snyk
snyk container test vulntor/vulntor:latest
```

## Performance Optimization

### Resource Limits

```bash
# Set CPU and memory limits
docker run --cpus=2 --memory=4g vulntor/vulntor scan 192.168.1.0/24

# CPU shares
docker run --cpu-shares=512 vulntor/vulntor scan 192.168.1.0/24

# Memory reservation
docker run --memory=4g --memory-reservation=2g vulntor/vulntor scan 192.168.1.0/24
```

In Docker Compose:

```yaml
services:
  vulntor:
    image: vulntor/vulntor
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
docker run --tmpfs /tmp:rw,size=1g vulntor/vulntor scan 192.168.1.0/24

# Use volumes instead of bind mounts
docker volume create vulntor-storage
docker run -v vulntor-storage:/storage vulntor/vulntor scan 192.168.1.0/24
```

## Monitoring and Logging

### Container Logs

```bash
# View logs
docker logs vulntor-server

# Follow logs
docker logs -f vulntor-server

# Last 100 lines
docker logs --tail 100 vulntor-server

# Since timestamp
docker logs --since 2024-10-06T10:00:00 vulntor-server
```

### Log Drivers

```bash
# JSON file (default)
docker run --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  vulntor/vulntor server start

# Syslog
docker run --log-driver syslog \
  --log-opt syslog-address=udp://logs.company.com:514 \
  vulntor/vulntor server start

# Fluentd
docker run --log-driver fluentd \
  --log-opt fluentd-address=localhost:24224 \
  vulntor/vulntor server start
```

### Health Checks

```bash
# Add health check
docker run --health-cmd="curl -f http://localhost:8080/health || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  vulntor/vulntor server start

# Check health status
docker inspect --format='{{.State.Health.Status}}' vulntor-server
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Security Scan with Vulntor

on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - name: Run Vulntor Scan
        run: |
          docker run --rm \
            -v $(pwd):/output \
            vulntor/vulntor:latest \
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
vulntor_scan:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker pull vulntor/vulntor:latest
    - docker run --rm vulntor/vulntor scan $SCAN_TARGETS -o results.json
  artifacts:
    paths:
      - results.json
```

## Troubleshooting

### Permission Denied

```bash
# Add capabilities
docker run --cap-add NET_RAW --cap-add NET_ADMIN vulntor/vulntor scan 192.168.1.0/24

# Run as root
docker run --user root vulntor/vulntor scan 192.168.1.0/24
```

### Network Not Reachable

```bash
# Use host network
docker run --network host vulntor/vulntor scan 192.168.1.0/24

# Check network connectivity
docker run --rm vulntor/vulntor ping 192.168.1.1
```

### Container Crashes

```bash
# Check logs
docker logs vulntor-server

# Inspect container
docker inspect vulntor-server

# Check resource limits
docker stats vulntor-server
```

## Next Steps

- [Standalone Deployment](/deployment/standalone) - CLI deployment
- [Server Mode Deployment](/deployment/server-mode) - Systemd service
- [Air-Gapped Deployment](/deployment/air-gapped) - Offline deployment
- [Enterprise Features](/enterprise/overview) - Scale and features
