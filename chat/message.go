package chat

import "time"

type Message struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
	User      string    `json:"user,omitempty"`
	Body      string    `json:"body,omitempty"`
	Rooms     []string  `json:"rooms,omitempty"`
	Users     []string  `json:"users,omitempty"`
	History   []Message `json:"history,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
