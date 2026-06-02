#!/usr/bin/env bash

# start-project.sh
# Usage: ./start-project.sh [localhost|docker]

set -euo pipefail

OPTION="${1:-docker}"

# ─── Colors ──────────────────────────────────────────────────────────────────

CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
DARK_GRAY='\033[0;90m'
WHITE='\033[0;37m'
NC='\033[0m'

write_done() { echo -e "${YELLOW}$1${NC}"; }
write_ok()   { echo -e "${GREEN}$1${NC}"; }
write_fail() { echo -e "${RED}$1${NC}"; }

# ─── Step runner ─────────────────────────────────────────────────────────────

invoke_step() {
    local label="$1"
    shift
    echo -e "${CYAN}${label}...${NC}"
    if ! "$@"; then
        write_fail "[FAILED] ${label}"
        exit 1
    fi
    write_ok "[OK] ${label}"
}

# ─── Checks ──────────────────────────────────────────────────────────────────

check_docker() {
    if ! docker info > /dev/null 2>&1; then
        write_fail "> Docker is not running. Please start Docker and try again."
        exit 1
    fi
    write_ok "> Docker is running [DONE]"
}

check_port() {
    local port_type="$1"
    local port_number="$2"
    if ss -tlnp 2>/dev/null | grep -q ":${port_number} " || \
       lsof -iTCP:"${port_number}" -sTCP:LISTEN > /dev/null 2>&1; then
        write_fail "> Port ${port_type} ${port_number} is already in use [FAILED]"
        ss -tlnp | grep ":${port_number} " || true
        exit 1
    fi
    write_ok "> Port ${port_type} ${port_number} is available [DONE]"
}

check_go() {
    if ! go_version=$(go version 2>&1); then
        write_fail "> Go is not installed. Please install Go 1.21 or higher."
        exit 1
    fi
    if [[ "$go_version" =~ go([0-9]+)\.([0-9]+) ]]; then
        major="${BASH_REMATCH[1]}"
        minor="${BASH_REMATCH[2]}"
        if [[ "$major" -lt 1 ]] || [[ "$major" -eq 1 && "$minor" -lt 21 ]]; then
            write_fail "> Go version is less than 1.21. Please update Go."
            exit 1
        fi
        write_ok "> Go ${major}.${minor} detected [DONE]"
    fi
}

import_env_file() {
    local env_file="$1"
    [[ -f "$env_file" ]] || return 0
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*([^#[:space:]][^=]*)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]// /}"
            local value="${BASH_REMATCH[2]// /}"
            export "$key=$value"
        fi
    done < "$env_file"
    write_ok "> Loaded env: ${env_file} [DONE]"
}

# ─── Main ────────────────────────────────────────────────────────────────────

check_docker
check_port "http"  9090
check_port "mongo" 27017

if [[ "$OPTION" == "localhost" ]]; then
    check_go

    invoke_step "Generating swagger" swag init -g cmd/main.go --output docs/

    echo -e "${CYAN}> Starting MongoDB container...${NC}"
    docker compose up mongo -d
    write_ok "> MongoDB started [DONE]"

    import_env_file ".env"

    echo ""
    echo -e "${GREEN}Starting API locally...${NC}"
    echo -e "${WHITE}   Swagger: http://localhost:9090/swagger/index.html${NC}"
    echo ""

    stop_mongo() {
        echo ""
        echo -e "${YELLOW}Stopping MongoDB container...${NC}"
        docker compose stop mongo
        echo -e "${GREEN}Done.${NC}"
    }
    trap stop_mongo EXIT

    go run ./cmd/main.go

else
    stop_containers() {
        echo ""
        echo -e "${YELLOW}Stopping containers...${NC}"
        docker compose stop
        echo -e "${GREEN}Done.${NC}"
    }
    trap stop_containers EXIT

    invoke_step "Generating swagger" swag init -g cmd/main.go --output docs/
    invoke_step "Building docker image" docker compose build
    invoke_step "Starting containers"  docker compose up -d

    echo ""
    echo -e "${GREEN}Project started!${NC}"
    echo -e "${WHITE}   Swagger: http://localhost:9090/swagger/index.html${NC}"
    echo ""
    echo -e "${DARK_GRAY}Streaming container logs (Ctrl+C to stop)...${NC}"
    echo ""

    docker compose logs -f
fi
