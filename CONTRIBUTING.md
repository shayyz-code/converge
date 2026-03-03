# Contributing

Thank you for your interest in contributing! This project welcomes issues, PRs, and ideas.

## Quick Start

1. Fork and clone the repo
2. Install Go 1.25+
3. Install deps: `go mod tidy`
4. Run server (SQLite):
   - `CONVERGE_DB_PATH=converge.db CONVERGE_JWT_SECRET=dev-secret go run ./cmd/server`
   - By default, the server listens on `0.0.0.0:8080`.
5. Run TUI client:
   - `go run ./cmd/client -server ws://<server-ip>:8080/ws -room lobby -token "$JWT_TOKEN"`

Postgres:
- `CONVERGE_DB_ADAPTER=postgres CONVERGE_DB_DSN="postgres://user:pass@localhost:5432/converge?sslmode=disable" CONVERGE_JWT_SECRET=dev-secret go run ./cmd/server`

## Development

- Use JWT with `user_id` (and optional `display_name`)
- CRDT messages are available (OR-Set, LWW). See README for payloads
- Run tests: `go test ./...`

## Style

- Follow existing patterns and naming
- Keep functions small and cohesive
- Avoid logging secrets

## Commit and PR

- Prefer small, focused PRs with clear descriptions
- Reference issues when relevant
- Include tests for new behavior

## Code of Conduct

- See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). By participating, you agree to abide by it.
