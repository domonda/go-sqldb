#!/bin/bash
#
# Reset the PostgreSQL data directory used by docker-compose.
# Run this after changing the PostgreSQL version in docker-compose.yml,
# because PostgreSQL data files are not compatible across major versions.
#
# Usage: ./reset-postgres-data.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "Stopping containers..."
docker compose down

echo "Removing postgres-data directory..."
rm -rf postgres-data

echo "Starting containers with fresh database..."
docker compose up -d

echo "Waiting for PostgreSQL to be ready..."
until docker compose exec -T postgres pg_isready -U testuser > /dev/null 2>&1; do
    sleep 1
done

echo "PostgreSQL is ready."
