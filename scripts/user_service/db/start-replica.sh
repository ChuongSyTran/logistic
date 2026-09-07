#!/bin/bash
set -e

echo "WAITING 10 SECOND FOR MASTER START SUCCESSFULLY"
sleep 10s

if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "Starting sync data from Master (pg_basebackup)..."
    pg_basebackup -h user-db-master -D "$PGDATA" -U "$REPLICATION_USER" -vP -w -R
else
    echo "The data already exists, skip the data scraping step."
fi

echo "Starting Replica Server..."
exec docker-entrypoint.sh postgres
