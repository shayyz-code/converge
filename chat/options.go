package chat

import "time"

type Options struct {
	AllowedOrigins  []string
	MaxMessageBytes int64
	MaxBodyLength   int
	MaxRoomLength   int
	MaxUserLength   int
	HistoryLimit    int
	SendBuffer      int
	SaveBuffer      int
	StoreTimeout    time.Duration
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
}

func normalizeOptions(opts Options) Options {
	if opts.MaxMessageBytes == 0 {
		opts.MaxMessageBytes = 64 * 1024
	}
	if opts.MaxBodyLength == 0 {
		opts.MaxBodyLength = 2000
	}
	if opts.MaxRoomLength == 0 {
		opts.MaxRoomLength = 64
	}
	if opts.MaxUserLength == 0 {
		opts.MaxUserLength = 64
	}
	if opts.HistoryLimit == 0 {
		opts.HistoryLimit = 200
	}
	if opts.SendBuffer == 0 {
		opts.SendBuffer = 16
	}
	if opts.SaveBuffer == 0 {
		opts.SaveBuffer = 256
	}
	if opts.StoreTimeout == 0 {
		opts.StoreTimeout = 2 * time.Second
	}
	if opts.WriteWait == 0 {
		opts.WriteWait = 10 * time.Second
	}
	if opts.PongWait == 0 {
		opts.PongWait = 60 * time.Second
	}
	if opts.PingPeriod == 0 {
		opts.PingPeriod = (opts.PongWait * 9) / 10
	}
	return opts
}
