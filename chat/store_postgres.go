package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	if dsn == "" {
		return nil, errors.New("postgres dsn required")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{pool: pool}
	if err := store.initSchema(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func (s *PostgresStore) SaveMessage(ctx context.Context, msg Message) error {
	ts := msg.Timestamp.UTC()
	_, err := s.pool.Exec(
		ctx,
		"INSERT INTO messages (id, type, room, user_id, body, ts) VALUES ($1, $2, $3, $4, $5, $6)",
		msg.ID,
		msg.Type,
		msg.Room,
		msg.UserID,
		msg.Body,
		ts,
	)
	return err
}

func (s *PostgresStore) ListMessages(ctx context.Context, room string, limit int) ([]Message, error) {
	limit = clampLimit(limit, 50, 200)
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, type, room, user_id, body, ts FROM messages WHERE room = $1 ORDER BY ts DESC LIMIT $2",
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
		var ts time.Time
		if err := rows.Scan(&msg.ID, &msg.Type, &msg.Room, &msg.UserID, &msg.Body, &ts); err != nil {
			return nil, err
		}
		msg.Timestamp = ts
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

func (s *PostgresStore) initSchema(ctx context.Context) error {
	_, err := s.pool.Exec(
		ctx,
		"CREATE TABLE IF NOT EXISTS messages (id TEXT PRIMARY KEY, type TEXT NOT NULL, room TEXT NOT NULL, user_id TEXT, body TEXT, ts TIMESTAMPTZ NOT NULL)",
	)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(
		ctx,
		"CREATE INDEX IF NOT EXISTS idx_messages_room_ts ON messages (room, ts)",
	)
	if err != nil {
		return err
	}
	return s.migrateUserID(ctx)
}

func (s *PostgresStore) migrateUserID(ctx context.Context) error {
	rows, err := s.pool.Query(
		ctx,
		"SELECT column_name FROM information_schema.columns WHERE table_name = 'messages'",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasUserID := false
	hasUserName := false
	hasUser := false
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		switch name {
		case "user_id":
			hasUserID = true
		case "user_name":
			hasUserName = true
		case "user":
			hasUser = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasUserID {
		if _, err := s.pool.Exec(ctx, "ALTER TABLE messages ADD COLUMN user_id TEXT"); err != nil {
			return err
		}
	}
	if hasUserName {
		_, err = s.pool.Exec(ctx, "UPDATE messages SET user_id = user_name WHERE user_id IS NULL AND user_name IS NOT NULL")
		if err != nil {
			return err
		}
	}
	if hasUser {
		_, err = s.pool.Exec(ctx, "UPDATE messages SET user_id = user WHERE user_id IS NULL AND user IS NOT NULL")
		if err != nil {
			return err
		}
	}
	return nil
}
