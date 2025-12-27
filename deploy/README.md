# Deploy Email API to a Fresh VM

This guide walks you through deploying the Email API to a fresh Ubuntu VM with SSL using Envoy Proxy.

## Prerequisites

- Fresh Ubuntu 22.04+ VM
- Domain pointed to your VM's IP (e.g., `api.simpleemailapi.dev`)
- SSH access to the VM

---

## Step 1: Initial VM Setup

SSH into your VM and run the setup script:

```bash
# Clone the repository
git clone https://github.com/your-repo/emailapi.git ~/emailapi
cd ~/emailapi/deploy

# Make scripts executable
chmod +x setup_vm.sh deploy.sh certbot-renew-hook.sh

# Run the setup script (installs Go, Redis, Envoy, Certbot)
./setup_vm.sh
```

> **Note:** Reboot or run `source /etc/profile` after setup to update PATH.

---

## Step 2: Configure Environment

```bash
# Copy and edit the environment file
cp .env.example .env
nano .env
```

Fill in your credentials:
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Usually `redis://localhost:6379`
- `AWS_*` - AWS credentials for SES/S3
- `CLERK_*` - Clerk authentication keys
- `TINYBIRD_*` - Tinybird analytics tokens

---

## Step 3: Obtain SSL Certificate

Before starting Envoy, obtain the SSL certificate using Certbot's standalone mode:

```bash
# Stop any service on port 80 if running
sudo systemctl stop envoy 2>/dev/null || true

# Obtain SSL certificate
sudo certbot certonly --standalone -d api.simpleemailapi.dev

# Install the renewal hook
sudo cp certbot-renew-hook.sh /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
```

Certbot will:
1. Verify domain ownership via HTTP-01 challenge
2. Obtain the certificate
3. Store certificates at `/etc/letsencrypt/live/api.simpleemailapi.dev/`
4. Set up auto-renewal with our reload hook

---

## Step 4: Setup Envoy Proxy

```bash
# Copy Envoy config
sudo cp ~/emailapi/deploy/envoy.yaml /etc/envoy/envoy.yaml

# Copy and enable Envoy service
sudo cp ~/emailapi/deploy/envoy.service /etc/systemd/system/envoy.service
sudo systemctl daemon-reload
sudo systemctl enable envoy

# Validate Envoy configuration
envoy --mode validate -c /etc/envoy/envoy.yaml

# Start Envoy
sudo systemctl start envoy

# Check status
sudo systemctl status envoy
```

---

## Step 5: Setup Systemd Service for API

```bash
# Copy service file
sudo cp emailapi.service /etc/systemd/system/

# Edit paths if needed (update WorkingDirectory, ExecStart, EnvironmentFile)
sudo nano /etc/systemd/system/emailapi.service

# Reload systemd and enable the service
sudo systemctl daemon-reload
sudo systemctl enable emailapi
```

---

## Step 6: Build and Start the API

```bash
# Navigate to the API directory
cd ~/emailapi/apps/api

# Download dependencies and build
go mod download
go build -o emailapi main.go

# Start the service
sudo systemctl start emailapi

# Check status
sudo systemctl status emailapi
```

---

## Step 7: Verify Deployment

```bash
# Check if the API is running (HTTP)
curl https://api.simpleemailapi.dev/health

# Test with grpcurl (native gRPC)
grpcurl api.simpleemailapi.dev:443 list

# Check service logs
sudo journalctl -u emailapi -f
```

---

## Updating the API

To deploy new changes:

```bash
cd ~/emailapi/deploy
./deploy.sh
```

This will pull latest changes, rebuild, and restart the service.

---

## Useful Commands

| Command | Description |
|---------|-------------|
| `sudo systemctl restart emailapi` | Restart the API |
| `sudo systemctl status emailapi` | Check API status |
| `sudo journalctl -u emailapi -f` | View API logs (live) |
| `sudo systemctl restart envoy` | Restart Envoy |
| `sudo systemctl reload envoy` | Reload Envoy config (SIGHUP) |
| `sudo systemctl status envoy` | Check Envoy status |
| `envoy --mode validate -c /etc/envoy/envoy.yaml` | Validate Envoy config |
| `curl localhost:9901/stats` | View Envoy statistics |
| `curl localhost:9901/clusters` | View upstream cluster health |
| `sudo certbot renew --dry-run` | Test SSL renewal |

---

## Troubleshooting

### API not starting
```bash
# Check logs for errors
sudo journalctl -u emailapi -n 50

# Verify environment file is correct
cat /path/to/.env
```

### Envoy errors
```bash
# Validate config first
envoy --mode validate -c /etc/envoy/envoy.yaml

# Check Envoy logs
sudo journalctl -u envoy -n 50

# Check admin interface
curl localhost:9901/server_info
```

### gRPC/Connect not working
```bash
# Test native gRPC
grpcurl api.simpleemailapi.dev:443 list

# Test Connect RPC (JSON)
curl -X POST https://api.simpleemailapi.dev/v1.UserService/GetCurrentUser \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{}'

# Check Envoy cluster health
curl localhost:9901/clusters | grep emailapi_backend
```

### SSL certificate issues
```bash
# Check certificate status
sudo certbot certificates

# Renew certificate manually
sudo systemctl stop envoy
sudo certbot renew
sudo systemctl start envoy

# Verify renewal hook
cat /etc/letsencrypt/renewal-hooks/deploy/envoy-reload.sh
```

### Streaming connections dropping
The Envoy configuration disables stream timeouts (`stream_idle_timeout: 0s`) to support long-lived streaming connections. Your API sends heartbeats every 25 seconds to keep connections alive.

If connections still drop:
```bash
# Check Envoy stats for timeouts
curl localhost:9901/stats | grep timeout

# Verify heartbeats are being sent
sudo journalctl -u emailapi | grep heartbeat
```

---

## Performance Tuning

The setup script automatically applies kernel optimizations for high-performance API workloads:

| Setting | Value | Purpose |
|---------|-------|---------|
| TCP Congestion | BBRv3 | Lower latency, fewer retransmits |
| Queue Discipline | fq | Fair queuing for BBR |
| Max Connections | 65535 | High concurrent connection handling |
| TCP Fast Open | Enabled | Faster connection establishment |
| Swappiness | 10 | Prefer RAM over swap |

### Verify Performance Settings

```bash
# Check BBR is active
sysctl net.ipv4.tcp_congestion_control

# View all custom settings
sysctl -a | grep -f /etc/sysctl.d/99-emailapi-performance.conf

# Check current connection limits
sysctl net.core.somaxconn
```

> **Note:** BBRv3 requires kernel 6.1+. Debian Bookworm (6.1 LTS) and Ubuntu 22.04+ with HWE kernel support BBRv3.

---

## Architecture

```
                    ┌─────────────────────────────────────┐
                    │            Internet                  │
                    └───────────────┬─────────────────────┘
                                    │
                    ┌───────────────▼─────────────────────┐
                    │     Envoy Proxy (Port 443/80)       │
                    │  - TLS Termination                  │
                    │  - HTTP/2 + gRPC + Connect RPC      │
                    │  - CORS handling                    │
                    │  - Health checks                    │
                    └───────────────┬─────────────────────┘
                                    │ HTTP/2 (h2c)
                    ┌───────────────▼─────────────────────┐
                    │     Go API (Port 8080)              │
                    │  - Connect RPC handlers             │
                    │  - Native gRPC support              │
                    │  - gRPC-Web support                 │
                    │  - Streaming with heartbeats        │
                    └─────────────────────────────────────┘
```

