package chat

import "context"

type Store interface {
	SaveMessage(ctx context.Context, msg Message) error
	ListMessages(ctx context.Context, room string, limit int) ([]Message, error)
}

type ClosableStore interface {
	Store
	Close() error
}

func clampLimit(limit int, fallback int, max int) int {
	if limit <= 0 {
		return fallback
	}
	if limit > max {
		return max
	}
	return limit
}
