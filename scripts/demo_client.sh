#!/bin/bash

# Default values
USER_ID=${1:-demo-user}
DISPLAY_NAME=${2:-"Demo User"}
SERVER_URL=${3:-${SERVER_URL:-ws://localhost:8080/ws}}
ROOM=${ROOM:-lobby}
export CONVERGE_JWT_SECRET=${CONVERGE_JWT_SECRET:-dev-secret}

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Generate token
echo "Generating demo token for user: $USER_ID ($DISPLAY_NAME)..."
JWT_TOKEN=$(go run "$SCRIPT_DIR/gen_token.go" "$USER_ID" "$DISPLAY_NAME")

if [ -z "$JWT_TOKEN" ]; then
    echo "Failed to generate token"
    exit 1
fi

echo "Connecting to $SERVER_URL in room $ROOM..."
go run ./cmd/client/main.go -server "$SERVER_URL" -room "$ROOM" -token "$JWT_TOKEN"
