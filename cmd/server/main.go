package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/shayyz-code/converge/chat"
)

func main() {
	dbPath := os.Getenv("CONVERGE_DB_PATH")
	store, err := chat.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	options := chat.Options{
		AllowedOrigins:  splitCSV(os.Getenv("CONVERGE_ALLOWED_ORIGINS")),
		MaxMessageBytes: readInt64Env("CONVERGE_MAX_MESSAGE_BYTES", 0),
		MaxBodyLength:   readIntEnv("CONVERGE_MAX_BODY_LENGTH", 0),
		MaxRoomLength:   readIntEnv("CONVERGE_MAX_ROOM_LENGTH", 0),
		MaxUserLength:   readIntEnv("CONVERGE_MAX_USER_LENGTH", 0),
		HistoryLimit:    readIntEnv("CONVERGE_HISTORY_LIMIT", 0),
		SendBuffer:      readIntEnv("CONVERGE_SEND_BUFFER", 0),
		SaveBuffer:      readIntEnv("CONVERGE_SAVE_BUFFER", 0),
		StoreTimeout:    readDurationEnv("CONVERGE_STORE_TIMEOUT", 0),
		WriteWait:       readDurationEnv("CONVERGE_WRITE_WAIT", 0),
		PongWait:        readDurationEnv("CONVERGE_PONG_WAIT", 0),
		PingPeriod:      readDurationEnv("CONVERGE_PING_PERIOD", 0),
	}
	hub := chat.NewHubWithOptions(store, options)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := readEnv("CONVERGE_ADDR", ":8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  readDurationEnv("CONVERGE_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: readDurationEnv("CONVERGE_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  readDurationEnv("CONVERGE_IDLE_TIMEOUT", 60*time.Second),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hub.Close()
	srv.Shutdown(shutdownCtx)
}

func readEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func readIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func readInt64Env(key string, fallback int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func readDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
