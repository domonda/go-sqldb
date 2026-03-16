#!/bin/bash
#
# Reset the MariaDB data directory used by docker-compose.
# Run this after changing the MariaDB version in docker-compose.yml,
# because MariaDB data files are not compatible across major versions.
#
# Usage: ./reset-mariadb-data.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "Stopping containers..."
docker compose down

echo "Removing mariadb-data directory..."
rm -rf mariadb-data

echo "Starting containers with fresh database..."
docker compose up -d

echo "Waiting for MariaDB to be ready..."
until docker compose exec -T mariadb mariadb-admin ping -u root -prootpassword --silent > /dev/null 2>&1; do
    sleep 1
done

echo "MariaDB is ready."
