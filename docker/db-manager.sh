#!/bin/bash

# JonBuhTrader Database Management Script
# This script helps manage the PostgreSQL TimescaleDB container

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$DOCKER_DIR")"
COMPOSE_FILE="$DOCKER_DIR/docker-compose.yml"

# Helper functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
}

# Start the database services
start_db() {
    print_info "Starting PostgreSQL TimescaleDB and PgAdmin..."
    cd "$DOCKER_DIR"
    docker-compose up -d
    print_success "Database services started!"
    print_info "TimescaleDB is available at: localhost:5432"
    print_info "PgAdmin is available at: http://localhost:8080"
    print_info "Default credentials are in the .env file"
}

# Stop the database services
stop_db() {
    print_info "Stopping database services..."
    cd "$DOCKER_DIR"
    docker-compose down
    print_success "Database services stopped!"
}

# Restart the database services
restart_db() {
    print_info "Restarting database services..."
    stop_db
    start_db
}

# Show database logs
logs_db() {
    cd "$DOCKER_DIR"
    if [ "$1" ]; then
        docker-compose logs -f "$1"
    else
        docker-compose logs -f
    fi
}

# Show database status
status_db() {
    cd "$DOCKER_DIR"
    docker-compose ps
}

# Connect to database
connect_db() {
    print_info "Connecting to TimescaleDB..."
    source "$DOCKER_DIR/.env"
    docker exec -it jonbuh_timescaledb psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
}

# Backup database
backup_db() {
    print_info "Creating database backup..."
    source "$DOCKER_DIR/.env"
    BACKUP_FILE="backup_$(date +%Y%m%d_%H%M%S).sql"
    docker exec jonbuh_timescaledb pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" > "$DOCKER_DIR/$BACKUP_FILE"
    print_success "Backup created: $DOCKER_DIR/$BACKUP_FILE"
}

# Reset database (WARNING: This will delete all data!)
reset_db() {
    print_warning "This will completely reset the database and delete ALL data!"
    read -p "Are you sure you want to continue? (type 'yes' to confirm): " confirm
    if [ "$confirm" = "yes" ]; then
        print_info "Stopping services and removing volumes..."
        cd "$DOCKER_DIR"
        docker-compose down -v
        print_info "Removing Docker volumes..."
        docker volume rm docker_timescale_data docker_pgadmin_data 2>/dev/null || true
        print_info "Starting fresh database..."
        docker-compose up -d
        print_success "Database has been reset!"
    else
        print_info "Reset cancelled."
    fi
}

# Show help
show_help() {
    cat << EOF
JonBuhTrader Database Management Script

Usage: $0 [COMMAND]

Commands:
    start       Start the database services
    stop        Stop the database services
    restart     Restart the database services
    status      Show status of database services
    logs        Show logs (optionally specify service: timescaledb or pgadmin)
    connect     Connect to the database via psql
    backup      Create a database backup
    reset       Reset the database (WARNING: Deletes all data!)
    help        Show this help message

Examples:
    $0 start
    $0 logs timescaledb
    $0 connect
    $0 backup

EOF
}

# Main script logic
check_docker

case "${1:-}" in
    "start")
        start_db
        ;;
    "stop")
        stop_db
        ;;
    "restart")
        restart_db
        ;;
    "status")
        status_db
        ;;
    "logs")
        logs_db "$2"
        ;;
    "connect")
        connect_db
        ;;
    "backup")
        backup_db
        ;;
    "reset")
        reset_db
        ;;
    "help"|"--help"|"-h")
        show_help
        ;;
    "")
        print_error "No command specified. Use 'help' to see available commands."
        show_help
        exit 1
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
