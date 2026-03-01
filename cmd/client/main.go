package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/shayyz-code/converge/chat"
)

type uiState struct {
	messages  []string
	input     string
	room      string
	user      string
	server    string
	connected bool
	scroll    int
	width     int
	height    int
}

func main() {
	server := flag.String("server", "ws://localhost:8080/ws", "")
	room := flag.String("room", "lobby", "")
	user := flag.String("user", "", "")
	flag.Parse()

	if *user == "" {
		*user = "user-" + strconv.FormatInt(time.Now().UnixNano()%100000, 10)
	}

	wsURL, err := buildWSURL(*server, *room, *user)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer screen.Fini()

	state := &uiState{
		room:      *room,
		user:      *user,
		server:    wsURL,
		connected: true,
	}

	eventCh := make(chan tcell.Event, 16)
	msgCh := make(chan chat.Message, 32)
	errCh := make(chan error, 1)
	sendCh := make(chan chat.Message, 32)
	done := make(chan struct{})

	go readLoop(conn, msgCh, errCh)
	go writeLoop(conn, sendCh, errCh)
	go pollEvents(screen, eventCh)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	pushMessage(state, "connected to "+wsURL)
	pushMessage(state, "type /help for commands")
	draw(screen, state)

	for {
		select {
		case ev := <-eventCh:
			if handleEvent(ev, state, sendCh) {
				return
			}
			draw(screen, state)
		case msg := <-msgCh:
			handleMessage(state, msg)
			draw(screen, state)
		case err := <-errCh:
			pushMessage(state, "error: "+err.Error())
			state.connected = false
			draw(screen, state)
			close(sendCh)
			return
		case <-sigCh:
			close(done)
			return
		case <-done:
			return
		}
	}
}

func buildWSURL(base, room, user string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := parsed.Query()
	if room != "" {
		q.Set("room", room)
	}
	if user != "" {
		q.Set("user", user)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

func readLoop(conn *websocket.Conn, msgCh chan<- chat.Message, errCh chan<- error) {
	for {
		var msg chat.Message
		if err := conn.ReadJSON(&msg); err != nil {
			errCh <- err
			return
		}
		msgCh <- msg
	}
}

func writeLoop(conn *websocket.Conn, sendCh <-chan chat.Message, errCh chan<- error) {
	for msg := range sendCh {
		if err := conn.WriteJSON(msg); err != nil {
			errCh <- err
			return
		}
	}
}

func pollEvents(screen tcell.Screen, eventCh chan<- tcell.Event) {
	for {
		ev := screen.PollEvent()
		eventCh <- ev
	}
}

func handleEvent(ev tcell.Event, state *uiState, sendCh chan<- chat.Message) bool {
	switch e := ev.(type) {
	case *tcell.EventResize:
		state.width, state.height = e.Size()
		state.scroll = 0
		return false
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyCtrlC:
			return true
		case tcell.KeyEnter:
			input := strings.TrimSpace(state.input)
			if input == "" {
				return false
			}
			state.input = ""
			handleInput(state, input, sendCh)
			return false
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if len(state.input) > 0 {
				state.input = string([]rune(state.input)[:len([]rune(state.input))-1])
			}
			return false
		case tcell.KeyUp:
			state.scroll++
			return false
		case tcell.KeyDown:
			if state.scroll > 0 {
				state.scroll--
			}
			return false
		case tcell.KeyPgUp:
			state.scroll += 5
			return false
		case tcell.KeyPgDn:
			if state.scroll > 5 {
				state.scroll -= 5
			} else {
				state.scroll = 0
			}
			return false
		}
		if e.Rune() != 0 {
			state.input += string(e.Rune())
		}
	}
	return false
}

func handleInput(state *uiState, input string, sendCh chan<- chat.Message) {
	if strings.HasPrefix(input, "/") {
		cmd := strings.TrimPrefix(input, "/")
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return
		}
		switch parts[0] {
		case "join":
			if len(parts) < 2 {
				pushMessage(state, "usage: /join room")
				return
			}
			state.room = parts[1]
			sendCh <- chat.Message{Type: "join", Room: parts[1]}
			pushMessage(state, "joining "+parts[1])
		case "rooms":
			sendCh <- chat.Message{Type: "rooms"}
		case "users":
			room := state.room
			if len(parts) > 1 {
				room = parts[1]
			}
			sendCh <- chat.Message{Type: "users", Room: room}
		case "history":
			limit := 50
			if len(parts) > 1 {
				if parsed, err := strconv.Atoi(parts[1]); err == nil {
					limit = parsed
				}
			}
			sendCh <- chat.Message{Type: "history", Room: state.room, Limit: limit}
		case "quit":
			sendCh <- chat.Message{Type: "leave"}
			os.Exit(0)
		case "help":
			pushMessage(state, "/join room")
			pushMessage(state, "/rooms")
			pushMessage(state, "/users [room]")
			pushMessage(state, "/history [limit]")
			pushMessage(state, "/quit")
		default:
			pushMessage(state, "unknown command: "+parts[0])
		}
		return
	}
	sendCh <- chat.Message{Type: "message", Room: state.room, Body: input}
}

func handleMessage(state *uiState, msg chat.Message) {
	switch msg.Type {
	case "history":
		for _, item := range msg.History {
			pushMessage(state, formatMessage(item))
		}
	default:
		pushMessage(state, formatMessage(msg))
	}
}

func formatMessage(msg chat.Message) string {
	timestamp := msg.Timestamp.Format("15:04:05")
	switch msg.Type {
	case "system":
		return "[" + timestamp + "] * " + msg.Body
	case "welcome":
		return "[" + timestamp + "] connected as " + msg.User + " in " + msg.Room
	case "rooms":
		return "[" + timestamp + "] rooms: " + strings.Join(msg.Rooms, ", ")
	case "users":
		return "[" + timestamp + "] users in " + msg.Room + ": " + strings.Join(msg.Users, ", ")
	case "error":
		return "[" + timestamp + "] error: " + msg.Body
	default:
		label := msg.User
		if label == "" {
			label = msg.Room
		}
		return "[" + timestamp + "] " + label + ": " + msg.Body
	}
}

func pushMessage(state *uiState, line string) {
	state.messages = append(state.messages, line)
	if len(state.messages) > 2000 {
		state.messages = state.messages[len(state.messages)-2000:]
	}
}

func draw(screen tcell.Screen, state *uiState) {
	screen.Clear()
	w, h := screen.Size()
	state.width, state.height = w, h

	status := fmt.Sprintf("room:%s user:%s connected:%t", state.room, state.user, state.connected)
	drawLine(screen, 0, 0, status, tcell.StyleDefault.Reverse(true))

	messageAreaHeight := h - 2
	lines := wrapMessages(state.messages, w)
	start := len(lines) - messageAreaHeight - state.scroll
	if start < 0 {
		start = 0
	}
	end := start + messageAreaHeight
	if end > len(lines) {
		end = len(lines)
	}
	row := 1
	for i := start; i < end; i++ {
		drawLine(screen, 0, row, lines[i], tcell.StyleDefault)
		row++
	}

	input := "> " + state.input
	drawLine(screen, 0, h-1, input, tcell.StyleDefault)
	screen.Show()
}

func drawLine(screen tcell.Screen, x, y int, text string, style tcell.Style) {
	width, _ := screen.Size()
	col := x
	for _, r := range text {
		if col >= width {
			break
		}
		screen.SetContent(col, y, r, nil, style)
		col++
	}
	for col < width {
		screen.SetContent(col, y, ' ', nil, style)
		col++
	}
}

func wrapMessages(messages []string, width int) []string {
	lines := []string{}
	for _, msg := range messages {
		lines = append(lines, wrapLine(msg, width)...)
	}
	return lines
}

func wrapLine(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	parts := []string{}
	runes := []rune(text)
	for len(runes) > 0 {
		chunk := width
		if len(runes) < width {
			chunk = len(runes)
		}
		parts = append(parts, string(runes[:chunk]))
		runes = runes[chunk:]
	}
	if len(parts) == 0 {
		return []string{""}
	}
	return parts
}
