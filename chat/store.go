package chat

import (
	"context"
	"database/sql"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store interface {
	SaveMessage(ctx context.Context, msg Message) error
	ListMessages(ctx context.Context, room string, limit int) ([]Message, error)
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dsn := sqliteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	store := &SQLiteStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) SaveMessage(ctx context.Context, msg Message) error {
	ts := msg.Timestamp.UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO messages (id, type, room, user, body, ts) VALUES (?, ?, ?, ?, ?, ?)",
		msg.ID,
		msg.Type,
		msg.Room,
		msg.User,
		msg.Body,
		ts,
	)
	return err
}

func (s *SQLiteStore) ListMessages(ctx context.Context, room string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT id, type, room, user, body, ts FROM messages WHERE room = ? ORDER BY ts DESC LIMIT ?",
		room,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []Message{}
	for rows.Next() {
		var msg Message
		var ts string
		if err := rows.Scan(&msg.ID, &msg.Type, &msg.Room, &msg.User, &msg.Body, &ts); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			parsed = time.Time{}
		}
		msg.Timestamp = parsed
		history = append(history, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	return history, nil
}

func (s *SQLiteStore) initSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(
		ctx,
		"CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, type TEXT NOT NULL, room TEXT NOT NULL, user TEXT, body TEXT, ts TEXT NOT NULL)",
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		"CREATE INDEX IF NOT EXISTS idx_messages_room_ts ON messages (room, ts)",
	)
	return err
}

func sqliteDSN(path string) string {
	if path == "" {
		path = "converge.db"
	}
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}
