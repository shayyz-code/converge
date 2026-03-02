package chat

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	send        chan Message
	room        string
	userID      string
	displayName string
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return h.isOriginAllowed(r.Header.Get("Origin"))
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	room := r.URL.Query().Get("room")
	room = strings.TrimSpace(room)
	if room == "" {
		room = "lobby"
	}
	token := extractToken(r)
	userID, displayName, err := h.authenticateToken(token)
	if err != nil {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"))
		conn.Close()
		return
	}
	if len(room) > h.options.MaxRoomLength || len(userID) > h.options.MaxUserLength || len(displayName) > h.options.MaxUserLength {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid room or user"))
		conn.Close()
		return
	}
	client := &Client{
		hub:         h,
		conn:        conn,
		send:        make(chan Message, h.options.SendBuffer),
		room:        room,
		userID:      userID,
		displayName: displayName,
	}
	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(c.hub.options.MaxMessageBytes)
	c.conn.SetReadDeadline(time.Now().Add(c.hub.options.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.hub.options.PongWait))
		return nil
	})
	for {
		var incoming Message
		if err := c.conn.ReadJSON(&incoming); err != nil {
			break
		}
		if incoming.Body != "" && len(incoming.Body) > c.hub.options.MaxBodyLength {
			c.send <- Message{
				ID:        uuid.NewString(),
				Type:      "error",
				Body:      "message too large",
				Timestamp: time.Now().UTC(),
			}
			continue
		}
		switch incoming.Type {
		case "join":
			if incoming.Room != "" && incoming.Room != c.room {
				c.hub.move <- roomMove{client: c, toRoom: incoming.Room}
			}
		case "leave":
			return
		case "rooms":
			c.hub.roomsReq <- roomsRequest{client: c}
		case "users":
			c.hub.usersReq <- usersRequest{client: c, room: incoming.Room}
		case "history":
			if c.hub.store == nil {
				c.send <- Message{
					ID:        uuid.NewString(),
					Type:      "error",
					Body:      "history unavailable",
					Timestamp: time.Now().UTC(),
				}
				continue
			}
			room := incoming.Room
			if room == "" {
				room = c.room
			}
			limit := incoming.Limit
			if limit <= 0 {
				limit = 50
			} else if limit > c.hub.options.HistoryLimit {
				limit = c.hub.options.HistoryLimit
			}
			ctx, cancel := context.WithTimeout(context.Background(), c.hub.options.StoreTimeout)
			history, err := c.hub.store.ListMessages(ctx, room, limit)
			cancel()
			if err != nil {
				c.send <- Message{
					ID:        uuid.NewString(),
					Type:      "error",
					Body:      "history unavailable",
					Timestamp: time.Now().UTC(),
				}
				continue
			}
			c.send <- Message{
				ID:        uuid.NewString(),
				Type:      "history",
				Room:      room,
				History:   history,
				Timestamp: time.Now().UTC(),
			}
		default:
			if incoming.Room != "" && incoming.Room != c.room {
				c.hub.move <- roomMove{client: c, toRoom: incoming.Room}
			}
			room := c.room
			out := Message{
				ID:          uuid.NewString(),
				Type:        "message",
				Room:        room,
				UserID:      c.userID,
				DisplayName: c.displayName,
				Body:        incoming.Body,
				Timestamp:   time.Now().UTC(),
			}
			c.hub.broadcast <- out
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(c.hub.options.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.options.WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.options.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
