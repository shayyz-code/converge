#!/bin/bash

# Default values
export CONVERGE_DB_PATH=${CONVERGE_DB_PATH:-converge.db}
export CONVERGE_JWT_SECRET=${CONVERGE_JWT_SECRET:-dev-secret}
export PORT=${PORT:-8080}
export CONVERGE_HOST=${CONVERGE_HOST:-0.0.0.0}

echo "Starting Converge server on ${CONVERGE_HOST}:${PORT}..."
echo "Database path: ${CONVERGE_DB_PATH}"

go run ./cmd/server/main.go
