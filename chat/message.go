package chat

import "time"

type Message struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Room        string    `json:"room,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Doc         string    `json:"doc,omitempty"`
	Body        string    `json:"body,omitempty"`
	Rooms       []string  `json:"rooms,omitempty"`
	Users       []string  `json:"users,omitempty"`
	Items       []string  `json:"items,omitempty"`
	History     []Message `json:"history,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
