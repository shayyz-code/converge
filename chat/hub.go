package chat

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shayyz-code/converge/chat/crdt"
)

type roomMove struct {
	client *Client
	toRoom string
}

type roomsRequest struct {
	client *Client
}

type usersRequest struct {
	client *Client
	room   string
}

type Hub struct {
	rooms             map[string]map[*Client]bool
	register          chan *Client
	unregister        chan *Client
	move              chan roomMove
	broadcast         chan Message
	roomsReq          chan roomsRequest
	usersReq          chan usersRequest
	store             Store
	save              chan Message
	options           Options
	allowedOrigins    map[string]bool
	allowAllOrigins   bool
	shutdownRequested chan struct{}
	crdtOps           chan crdtOp
	crdtOrSets        map[string]*crdt.ORSet[string]
	crdtLWW           map[string]crdt.LWWRegister[string]
}

type crdtOp struct {
	kind   string
	action string
	room   string
	doc    string
	value  string
	node   string
	client *Client
}

func NewHub(store Store) *Hub {
	return NewHubWithOptions(store, Options{})
}

func NewHubWithOptions(store Store, options Options) *Hub {
	opts := normalizeOptions(options)
	allowedOrigins := map[string]bool{}
	allowAll := len(opts.AllowedOrigins) == 0
	for _, origin := range opts.AllowedOrigins {
		if origin == "*" {
			allowAll = true
			continue
		}
		allowedOrigins[origin] = true
	}
	return &Hub{
		rooms:             make(map[string]map[*Client]bool),
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		move:              make(chan roomMove),
		broadcast:         make(chan Message, 64),
		roomsReq:          make(chan roomsRequest),
		usersReq:          make(chan usersRequest),
		store:             store,
		save:              make(chan Message, opts.SaveBuffer),
		options:           opts,
		allowedOrigins:    allowedOrigins,
		allowAllOrigins:   allowAll,
		shutdownRequested: make(chan struct{}),
		crdtOps:           make(chan crdtOp, 128),
		crdtOrSets:        make(map[string]*crdt.ORSet[string]),
		crdtLWW:           make(map[string]crdt.LWWRegister[string]),
	}
}

func (h *Hub) Run() {
	if h.store != nil {
		go h.runStore()
	}
	for {
		select {
		case <-h.shutdownRequested:
			h.closeAllClients()
			close(h.save)
			return
		case client := <-h.register:
			room := client.room
			if room == "" {
				room = "lobby"
				client.room = room
			}
			if h.rooms[room] == nil {
				h.rooms[room] = make(map[*Client]bool)
			}
			h.rooms[room][client] = true
			h.sendToClient(client, Message{
				ID:          uuid.NewString(),
				Type:        "welcome",
				Room:        room,
				UserID:      client.userID,
				DisplayName: client.displayName,
				Timestamp:   time.Now().UTC(),
			})
			h.broadcast <- Message{
				ID:        uuid.NewString(),
				Type:      "system",
				Room:      room,
				Body:      client.displayName + " joined",
				Timestamp: time.Now().UTC(),
			}
		case client := <-h.unregister:
			room := client.room
			if roomClients, ok := h.rooms[room]; ok {
				delete(roomClients, client)
				if len(roomClients) == 0 {
					delete(h.rooms, room)
				}
			}
			h.closeClient(client)
			h.broadcast <- Message{
				ID:        uuid.NewString(),
				Type:      "system",
				Room:      room,
				Body:      client.displayName + " left",
				Timestamp: time.Now().UTC(),
			}
		case move := <-h.move:
			if move.toRoom == "" {
				continue
			}
			if move.client.room == move.toRoom {
				continue
			}
			if roomClients, ok := h.rooms[move.client.room]; ok {
				delete(roomClients, move.client)
				if len(roomClients) == 0 {
					delete(h.rooms, move.client.room)
				}
			}
			if h.rooms[move.toRoom] == nil {
				h.rooms[move.toRoom] = make(map[*Client]bool)
			}
			h.rooms[move.toRoom][move.client] = true
			oldRoom := move.client.room
			move.client.room = move.toRoom
			h.broadcast <- Message{
				ID:        uuid.NewString(),
				Type:      "system",
				Room:      oldRoom,
				Body:      move.client.displayName + " left",
				Timestamp: time.Now().UTC(),
			}
			h.broadcast <- Message{
				ID:        uuid.NewString(),
				Type:      "system",
				Room:      move.toRoom,
				Body:      move.client.displayName + " joined",
				Timestamp: time.Now().UTC(),
			}
		case msg := <-h.broadcast:
			if msg.Room == "" {
				continue
			}
			if roomClients, ok := h.rooms[msg.Room]; ok {
				for client := range roomClients {
					select {
					case client.send <- msg:
					default:
						close(client.send)
						delete(roomClients, client)
					}
				}
				if len(roomClients) == 0 {
					delete(h.rooms, msg.Room)
				}
			}
			if h.store != nil && (msg.Type == "message" || msg.Type == "system") {
				select {
				case h.save <- msg:
				default:
				}
			}
		case req := <-h.roomsReq:
			rooms := make([]string, 0, len(h.rooms))
			for room := range h.rooms {
				rooms = append(rooms, room)
			}
			req.client.send <- Message{
				ID:        uuid.NewString(),
				Type:      "rooms",
				Rooms:     rooms,
				Timestamp: time.Now().UTC(),
			}
		case req := <-h.usersReq:
			room := req.room
			if room == "" {
				room = req.client.room
			}
			users := []string{}
			if roomClients, ok := h.rooms[room]; ok {
				for client := range roomClients {
					users = append(users, client.userID)
				}
			}
			req.client.send <- Message{
				ID:        uuid.NewString(),
				Type:      "users",
				Room:      room,
				Users:     users,
				Timestamp: time.Now().UTC(),
			}
		case op := <-h.crdtOps:
			switch op.kind {
			case "orset":
				key := op.room + ":" + op.doc
				set := h.crdtOrSets[key]
				if set == nil {
					set = crdt.NewORSet[string](op.node)
					h.crdtOrSets[key] = set
				}
				switch op.action {
				case "add":
					set.Add(op.value)
					h.broadcast <- Message{
						ID:          uuid.NewString(),
						Type:        "crdt_orset_added",
						Room:        op.room,
						Doc:         op.doc,
						UserID:      op.node,
						DisplayName: op.client.displayName,
						Body:        op.value,
						Timestamp:   time.Now().UTC(),
					}
				case "remove":
					set.Remove(op.value)
					h.broadcast <- Message{
						ID:          uuid.NewString(),
						Type:        "crdt_orset_removed",
						Room:        op.room,
						Doc:         op.doc,
						UserID:      op.node,
						DisplayName: op.client.displayName,
						Body:        op.value,
						Timestamp:   time.Now().UTC(),
					}
				case "values":
					items := set.Values()
					op.client.send <- Message{
						ID:        uuid.NewString(),
						Type:      "crdt_orset_values",
						Room:      op.room,
						Doc:       op.doc,
						Items:     items,
						Timestamp: time.Now().UTC(),
					}
				}
			case "lww":
				key := op.room + ":" + op.doc
				reg := h.crdtLWW[key]
				switch op.action {
				case "set":
					if reg.Node == "" && reg.Timestamp.IsZero() {
						reg = crdt.NewLWWRegister(op.value, time.Now().UTC(), op.node)
					} else {
						reg.Set(op.value, time.Now().UTC(), op.node)
					}
					h.crdtLWW[key] = reg
					h.broadcast <- Message{
						ID:          uuid.NewString(),
						Type:        "crdt_lww_set",
						Room:        op.room,
						Doc:         op.doc,
						UserID:      op.node,
						DisplayName: op.client.displayName,
						Body:        reg.Value,
						Timestamp:   time.Now().UTC(),
					}
				case "get":
					op.client.send <- Message{
						ID:        uuid.NewString(),
						Type:      "crdt_lww_value",
						Room:      op.room,
						Doc:       op.doc,
						Body:      reg.Value,
						Timestamp: time.Now().UTC(),
					}
				}
			}
		}
	}
}

func (h *Hub) runStore() {
	for msg := range h.save {
		ctx, cancel := context.WithTimeout(context.Background(), h.options.StoreTimeout)
		_ = h.store.SaveMessage(ctx, msg)
		cancel()
	}
}

func (h *Hub) Close() {
	select {
	case <-h.shutdownRequested:
		return
	default:
		close(h.shutdownRequested)
	}
}

func (h *Hub) sendToClient(client *Client, msg Message) {
	select {
	case client.send <- msg:
	default:
		h.closeClient(client)
	}
}

func (h *Hub) closeClient(client *Client) {
	defer func() {
		recover()
	}()
	close(client.send)
	client.conn.Close()
}

func (h *Hub) closeAllClients() {
	for _, roomClients := range h.rooms {
		for client := range roomClients {
			h.closeClient(client)
		}
	}
	h.rooms = make(map[string]map[*Client]bool)
}

func (h *Hub) isOriginAllowed(origin string) bool {
	if h.allowAllOrigins {
		return true
	}
	if origin == "" {
		return false
	}
	return h.allowedOrigins[origin]
}
