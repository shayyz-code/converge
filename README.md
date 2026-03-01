# Converge

[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)

Real-time websocket chat server with SQLite/Postgres persistence and a built-in TUI client.

## Features

- Multi-room websocket chat with join/leave events
- Message persistence in SQLite with history retrieval
- Room and user listing
- Configurable limits and timeouts
- TUI client for local testing and demos

## Requirements

- Go 1.25+

## Run the server

```bash
CONVERGE_DB_PATH=converge.db CONVERGE_JWT_SECRET=dev-secret go run ./cmd/server
```

Postgres:

```bash
CONVERGE_DB_ADAPTER=postgres CONVERGE_DB_DSN="postgres://user:pass@localhost:5432/converge?sslmode=disable" CONVERGE_JWT_SECRET=dev-secret go run ./cmd/server
```

Health check:

```bash
curl http://localhost:8080/health
```

Websocket URL:

```
ws://localhost:8080/ws?room=lobby
```

Authentication:

- Send `Authorization: Bearer <jwt>`
- Token must include `user_id` or `sub`

## Run the TUI client

```bash
go run ./cmd/client -server ws://localhost:8080/ws -room lobby -token "$JWT_TOKEN"
```

### TUI commands

- `/join room`
- `/rooms`
- `/users [room]`
- `/history [limit]`
- `/quit`

## Protocol

### Client → Server

- Send message

```json
{ "type": "message", "body": "hello" }
```

- Join room

```json
{ "type": "join", "room": "dev" }
```

- List rooms

```json
{ "type": "rooms" }
```

- List users

```json
{ "type": "users", "room": "dev" }
```

- Fetch history

```json
{ "type": "history", "room": "dev", "limit": 50 }
```

### Server → Client

- Welcome

```json
{
  "type": "welcome",
  "room": "lobby",
  "user_id": "user-123",
  "timestamp": "..."
}
```

- System event

```json
{
  "type": "system",
  "room": "lobby",
  "body": "shayy joined",
  "timestamp": "..."
}
```

- Rooms list

```json
{ "type": "rooms", "rooms": ["lobby", "dev"], "timestamp": "..." }
```

- Users list

```json
{
  "type": "users",
  "room": "lobby",
  "users": ["user-123", "user-456"],
  "timestamp": "..."
}
```

- History

```json
{"type":"history","room":"lobby","history":[...],"timestamp":"..."}
```

- Error

```json
{ "type": "error", "body": "message too large", "timestamp": "..." }
```

## Configuration

| Env Var                    | Description                    | Default           |
| -------------------------- | ------------------------------ | ----------------- |
| CONVERGE_ADDR              | HTTP listen address            | :8080             |
| CONVERGE_DB_ADAPTER        | sqlite or postgres             | sqlite            |
| CONVERGE_DB_PATH           | SQLite file path               | converge.db       |
| CONVERGE_DB_DSN            | Postgres connection string     | empty             |
| CONVERGE_ALLOWED_ORIGINS   | Comma-separated origins or `*` | empty (allow all) |
| CONVERGE_MAX_MESSAGE_BYTES | Max websocket frame size       | 65536             |
| CONVERGE_MAX_BODY_LENGTH   | Max chat message length        | 2000              |
| CONVERGE_MAX_ROOM_LENGTH   | Max room name length           | 64                |
| CONVERGE_MAX_USER_LENGTH   | Max user name length           | 64                |
| CONVERGE_HISTORY_LIMIT     | Max history limit              | 200               |
| CONVERGE_SEND_BUFFER       | Per-client send buffer size    | 16                |
| CONVERGE_SAVE_BUFFER       | Persist queue size             | 256               |
| CONVERGE_STORE_TIMEOUT     | Store operation timeout        | 2s                |
| CONVERGE_WRITE_WAIT        | Websocket write deadline       | 10s               |
| CONVERGE_PONG_WAIT         | Websocket pong wait            | 60s               |
| CONVERGE_PING_PERIOD       | Websocket ping period          | 54s               |
| CONVERGE_READ_TIMEOUT      | HTTP read timeout              | 10s               |
| CONVERGE_WRITE_TIMEOUT     | HTTP write timeout             | 10s               |
| CONVERGE_IDLE_TIMEOUT      | HTTP idle timeout              | 60s               |
| CONVERGE_JWT_SECRET        | HMAC secret for JWT            | empty             |
| CONVERGE_JWT_ISSUER        | JWT issuer                     | empty             |
| CONVERGE_JWT_AUDIENCE      | JWT audience                   | empty             |

## Tests

```bash
go test ./...
```

## Use as a package

```go
store, err := chat.NewSQLiteStore("converge.db")
if err != nil {
    panic(err)
}
defer store.Close()

hub := chat.NewHubWithOptions(store, chat.Options{
    JWTSecret: "dev-secret",
})
go hub.Run()

http.HandleFunc("/ws", hub.HandleWS)
```

Postgres adapter:

```go
store, err := chat.NewPostgresStore("postgres://user:pass@localhost:5432/converge?sslmode=disable")
if err != nil {
    panic(err)
}
defer store.Close()

hub := chat.NewHubWithOptions(store, chat.Options{
    JWTSecret: "dev-secret",
})
go hub.Run()
```

## Contributors

<a href="https://github.com/shayyz-code/converge/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=shayyz-code/converge" />
</a>

## License

MIT. See [LICENSE](LICENSE).
