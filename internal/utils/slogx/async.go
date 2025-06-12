package slogx

import (
	"context"
	"log/slog"
)

type log struct {
	level slog.Level
	msg   string
	args  []any
}

type AsyncSlog struct {
	logger *slog.Logger
	ctx    context.Context
	ch     chan *log
}

func NewAsyncSlog(ctx context.Context, logger *slog.Logger) *AsyncSlog {
	l := &AsyncSlog{
		logger: logger,
		ctx:    ctx,
		ch:     make(chan *log, 256),
	}
	go l.listen()
	return l
}

func (l *AsyncSlog) listen() {
	for {
		select {
		case <-l.ctx.Done():
			close(l.ch)
			return
		case data := <-l.ch:
			l.logger.Log(l.ctx, data.level, data.msg, data.args...)
		}
	}
}

func (l *AsyncSlog) Log(level slog.Level, msg string, args ...any) {
	l.ch <- &log{level: level, msg: msg, args: args}
}
