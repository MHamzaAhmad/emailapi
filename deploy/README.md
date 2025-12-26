# Deploy Email API to a Fresh VM

This guide walks you through deploying the Email API to a fresh Ubuntu VM with SSL.

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
chmod +x setup_vm.sh deploy.sh

# Run the setup script (installs Go, Redis, Nginx, Certbot)
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

## Step 3: Setup Systemd Service

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

## Step 4: Build and Start the API

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

## Step 5: Configure Nginx

```bash
# Copy nginx config
sudo cp ~/emailapi/deploy/nginx.conf /etc/nginx/sites-available/emailapi

# Create symlink
sudo ln -s /etc/nginx/sites-available/emailapi /etc/nginx/sites-enabled/

# Remove default site (optional)
sudo rm /etc/nginx/sites-enabled/default

# Test nginx config
sudo nginx -t
```

---

## Step 6: Add SSL with Certbot

```bash
# Obtain SSL certificate
sudo certbot --nginx -d api.simpleemailapi.dev
```

Certbot will:
1. Verify domain ownership
2. Obtain the certificate
3. Auto-configure Nginx SSL settings
4. Set up auto-renewal

```bash
# Restart nginx to apply changes
sudo systemctl restart nginx
```

---

## Step 7: Verify Deployment

```bash
# Check if the API is running
curl https://api.simpleemailapi.dev/health

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
| `sudo systemctl restart nginx` | Restart Nginx |
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

### Nginx errors
```bash
# Test config
sudo nginx -t

# Check nginx logs
sudo tail -f /var/log/nginx/error.log
```

### SSL certificate issues
```bash
# Renew certificate manually
sudo certbot renew

# Check certificate status
sudo certbot certificates
```
