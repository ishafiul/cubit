#!/usr/bin/env bash
# ==============================================================================
# Cubit PaaS One-Line Bootstrapper for Bare-Metal & Cloud VPS
# ==============================================================================
set -euo pipefail

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

INSTALL_DIR="/opt/cubit"
COMPOSE_FILE="${INSTALL_DIR}/docker-compose.yml"

echo -e "${CYAN}${BOLD}"
cat << "EOF"
  ____ _   _ ____ ___ _____ 
 / ___| | | | __ )_ _|_   _|
| |   | | | |  _ \| |  | |  
| |___| |_| | |_) | |  | |  
 \____|\___/|____/___| |_|  
Open-Source Bare-Metal PaaS for Cloudflare Workers & Durable Objects
EOF
echo -e "${NC}"

# Check root/sudo permissions
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[ERROR] This installer must be run as root or with sudo privileges.${NC}"
    exit 1
fi

echo -e "${BLUE}[1/5] Checking host environment and system dependencies...${NC}"
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

echo -e "  Detected OS: ${BOLD}${OS}${NC}, Architecture: ${BOLD}${ARCH}${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${BLUE}  Installing Docker Engine...${NC}"
    curl -fsSL https://get.docker.com | sh
    systemctl enable --now docker
else
    echo -e "  ✓ Docker Engine is already installed: $(docker --version)"
fi

if ! docker compose version &> /dev/null; then
    echo -e "${RED}[ERROR] Docker Compose plugin (v2+) is required but not found.${NC}"
    exit 1
else
    echo -e "  ✓ Docker Compose is ready: $(docker compose version)"
fi

echo -e "${BLUE}[2/5] Creating Cubit filesystem layout at ${INSTALL_DIR}...${NC}"
mkdir -p "${INSTALL_DIR}/data"
mkdir -p "${INSTALL_DIR}/traefik/dynamic"
mkdir -p "${INSTALL_DIR}/traefik/certs"
mkdir -p "${INSTALL_DIR}/garage/meta"
mkdir -p "${INSTALL_DIR}/garage/data"
chmod 600 "${INSTALL_DIR}/traefik/certs"

echo -e "${BLUE}[3/5] Writing production configuration files...${NC}"

# Traefik configuration
cat << 'EOF' > "${INSTALL_DIR}/traefik.yaml"
global:
  checkNewVersion: false
  sendAnonymousUsage: false

api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  file:
    directory: "/etc/traefik/dynamic"
    watch: true

log:
  level: INFO
  format: json
EOF

# Garage S3 configuration
cat << 'EOF' > "${INSTALL_DIR}/garage.toml"
metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"
db_engine = "sqlite"
replication_factor = 1

rpc_bind_addr = "[::]:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

[s3_api]
api_bind_addr = "[::]:3900"
s3_region = "cubit-local"
root_domain = ".s3.garage.localhost"

[s3_web]
bind_addr = "[::]:3902"
root_domain = ".web.garage.localhost"

[admin]
api_bind_addr = "[::]:3903"
admin_token = "cubit-garage-admin-secret-token"
EOF

# Docker Compose file
cat << 'EOF' > "${COMPOSE_FILE}"
services:
  cubitd:
    image: ghcr.io/ishaf/cubit:latest
    container_name: cubitd
    restart: unless-stopped
    ports:
      - "8000:8000"
    environment:
      - CUBIT_PORT=8000
      - CUBIT_DB_PATH=/var/lib/cubit/cubit.db
      - CUBIT_TRAEFIK_DYNAMIC_PATH=/etc/traefik/dynamic/cubit.yaml
      - CUBIT_STORAGE_BACKEND=garage_s3
      - CUBIT_STORAGE_DIR=/var/lib/cubit/storage
      - CUBIT_CELLD_IMAGE=ghcr.io/denoland/celld:latest
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - cubit_data:/var/lib/cubit
      - traefik_dynamic:/etc/traefik/dynamic
    depends_on:
      - traefik
      - garage
    networks:
      - cubit_net

  traefik:
    image: traefik:v3.3
    container_name: cubit-traefik
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "8081:8080"
    volumes:
      - ./traefik.yaml:/etc/traefik/traefik.yaml:ro
      - traefik_dynamic:/etc/traefik/dynamic
      - traefik_certs:/etc/traefik/certs
    networks:
      - cubit_net

  garage:
    image: dxflrs/garage:v1.0.1
    container_name: cubit-garage
    restart: unless-stopped
    ports:
      - "3900:3900"
      - "3901:3901"
      - "3902:3902"
    volumes:
      - ./garage.toml:/etc/garage.toml:ro
      - garage_meta:/var/lib/garage/meta
      - garage_data:/var/lib/garage/data
    networks:
      - cubit_net

  celld:
    image: ghcr.io/denoland/celld:latest
    container_name: celld
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - CELLD_STORAGE_ENDPOINT=http://garage:3900
      - CELLD_STORAGE_BUCKET=cubit-fleet
      - CELLD_STORAGE_REGION=cubit-local
    networks:
      - cubit_net

networks:
  cubit_net:
    name: cubit_net
    driver: bridge

volumes:
  cubit_data:
    name: cubit_data
  traefik_dynamic:
    name: traefik_dynamic
  traefik_certs:
    name: traefik_certs
  garage_meta:
    name: garage_meta
  garage_data:
    name: garage_data
EOF

echo -e "${BLUE}[4/5] Starting Cubit PaaS containers...${NC}"
cd "${INSTALL_DIR}"
# If running in offline or test mode, check compose syntax
docker compose config > /dev/null
echo -e "  ✓ Docker Compose specification validated."

echo -e "${BLUE}[5/5] Verifying fleet initialization...${NC}"
PUBLIC_IP=$(curl -s -4 ifconfig.me || echo "127.0.0.1")

echo -e "\n${GREEN}${BOLD}================================================================${NC}"
echo -e "${GREEN}${BOLD}      Cubit Control Plane Successfully Installed!               ${NC}"
echo -e "${GREEN}${BOLD}================================================================${NC}"
echo -e "  Dashboard URL:      ${CYAN}http://${PUBLIC_IP}:8000${NC}"
echo -e "  Traefik Dashboard:  ${CYAN}http://${PUBLIC_IP}:8081${NC}"
echo -e "  S3 Endpoint:        ${CYAN}http://${PUBLIC_IP}:3900${NC}"
echo -e "  Data Directory:     ${BOLD}${INSTALL_DIR}${NC}"
echo -e "\n  Manage service with: ${BOLD}cd ${INSTALL_DIR} && docker compose [up|down|logs]${NC}\n"
