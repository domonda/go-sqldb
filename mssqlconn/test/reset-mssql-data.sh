#!/bin/bash
#
# Reset the SQL Server data directory used by docker-compose.
# Run this after changing the SQL Server version in docker-compose.yml.
#
# Usage: ./reset-mssql-data.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "Stopping containers..."
docker compose down

echo "Removing mssql-data directory..."
rm -rf mssql-data

echo "Starting containers with fresh database..."
docker compose up -d

echo "Waiting for SQL Server to be ready..."
for i in $(seq 1 60); do
    if docker compose exec -T mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P 'TestPass123!' -C -Q "SELECT 1" > /dev/null 2>&1 || \
       docker compose exec -T mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P 'TestPass123!' -Q "SELECT 1" > /dev/null 2>&1; then
        echo "SQL Server is ready."
        exit 0
    fi
    sleep 2
done

echo "Warning: could not verify SQL Server readiness via sqlcmd."
echo "The server may still be starting. Run the integration tests to check connectivity."
