package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

func testToken(t *testing.T, secret, userID, display string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":      userID,
		"display_name": display,
		"iss":          "test",
		"aud":          "test-aud",
		"exp":          time.Now().Add(5 * time.Minute).Unix(),
		"iat":          time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func startTestServer(t *testing.T) (*Hub, *httptest.Server, func()) {
	t.Helper()
	store, err := NewSQLiteStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	options := Options{
		JWTSecret:       "secret",
		JWTIssuer:       "test",
		JWTAudience:     "test-aud",
		MaxMessageBytes: 64 * 1024,
	}
	hub := NewHubWithOptions(store, options)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	srv := httptest.NewServer(mux)
	cleanup := func() {
		hub.Close()
		srv.Close()
		_ = store.Close()
	}
	return hub, srv, cleanup
}

func wsURL(t *testing.T, httpURL string, room string) string {
	t.Helper()
	u, err := url.Parse(httpURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = "/ws"
	q := u.Query()
	q.Set("room", room)
	u.RawQuery = q.Encode()
	return u.String()
}

func dialWS(t *testing.T, uri, token string) (*websocket.Conn, func()) {
	t.Helper()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(uri, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	cleanup := func() {
		_ = conn.Close()
	}
	return conn, cleanup
}

func readUntilType(t *testing.T, conn *websocket.Conn, want string, timeout time.Duration) Message {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			continue
		}
		if msg.Type == want {
			return msg
		}
	}
	t.Fatalf("did not receive %s in time", want)
	return Message{}
}

func TestWebsocketWelcomeAndBroadcast(t *testing.T) {
	hub, srv, cleanup := startTestServer(t)
	defer cleanup()

	room := "test"
	uri := wsURL(t, srv.URL, room)
	secret := "secret"

	tok1 := testToken(t, secret, "u1", "Alice")
	c1, c1close := dialWS(t, uri, tok1)
	defer c1close()
	msg := readUntilType(t, c1, "welcome", 2*time.Second)
	if msg.UserID != "u1" || msg.DisplayName != "Alice" || msg.Room != room {
		t.Fatalf("welcome mismatch: %#v", msg)
	}

	tok2 := testToken(t, secret, "u2", "Bob")
	c2, c2close := dialWS(t, uri, tok2)
	defer c2close()
	_ = readUntilType(t, c2, "welcome", 2*time.Second)

	out := Message{Type: "message", Body: "hello from u1"}
	if err := c1.WriteJSON(out); err != nil {
		t.Fatalf("send: %v", err)
	}
	got := readUntilType(t, c2, "message", 2*time.Second)
	if got.Body != "hello from u1" || got.UserID != "u1" {
		t.Fatalf("broadcast mismatch: %#v", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	history, err := hub.store.ListMessages(ctx, room, 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(history) == 0 {
		t.Fatalf("expected persisted message")
	}
	found := false
	for _, m := range history {
		if m.Body == "hello from u1" && m.UserID == "u1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("message not found in store: %#v", history)
	}
}

func TestCRDTOpsOverWebsocket(t *testing.T) {
	_, srv, cleanup := startTestServer(t)
	defer cleanup()

	room := "crdt"
	uri := wsURL(t, srv.URL, room)
	secret := "secret"

	c1, close1 := dialWS(t, uri, testToken(t, secret, "u1", "Alice"))
	defer close1()
	_ = readUntilType(t, c1, "welcome", 2*time.Second)

	c2, close2 := dialWS(t, uri, testToken(t, secret, "u2", "Bob"))
	defer close2()
	_ = readUntilType(t, c2, "welcome", 2*time.Second)

	if err := c1.WriteJSON(Message{Type: "crdt_orset_add", Doc: "tags", Body: "alpha"}); err != nil {
		t.Fatalf("orset add: %v", err)
	}
	added := readUntilType(t, c2, "crdt_orset_added", 2*time.Second)
	if added.Doc != "tags" || added.Body != "alpha" {
		t.Fatalf("orset added mismatch: %#v", added)
	}

	if err := c1.WriteJSON(Message{Type: "crdt_orset_values", Doc: "tags"}); err != nil {
		t.Fatalf("orset values: %v", err)
	}
	vals := readUntilType(t, c1, "crdt_orset_values", 2*time.Second)
	found := false
	for _, it := range vals.Items {
		if it == "alpha" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("orset values missing alpha: %#v", vals.Items)
	}

	if err := c1.WriteJSON(Message{Type: "crdt_lww_set", Doc: "topic", Body: "Hello"}); err != nil {
		t.Fatalf("lww set: %v", err)
	}
	set := readUntilType(t, c2, "crdt_lww_set", 2*time.Second)
	if set.Doc != "topic" || set.Body != "Hello" {
		t.Fatalf("lww set mismatch: %#v", set)
	}
	if err := c2.WriteJSON(Message{Type: "crdt_lww_get", Doc: "topic"}); err != nil {
		t.Fatalf("lww get: %v", err)
	}
	got := readUntilType(t, c2, "crdt_lww_value", 2*time.Second)
	if got.Body != "Hello" {
		t.Fatalf("lww value mismatch: %#v", got)
	}
}
