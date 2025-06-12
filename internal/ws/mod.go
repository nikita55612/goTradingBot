package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// connection - WebSocket соединение с поддержкой переподключения.
type connection struct {
	conn         *websocket.Conn
	dialer       websocket.Dialer
	header       http.Header
	outChan      chan []byte
	reconnect    chan bool
	ctx          context.Context
	wg           sync.WaitGroup
	writeWait    time.Duration
	pongWait     time.Duration
	pingInterval time.Duration
	handshake    []byte
}

// Connect создает новое WebSocket соединение.
func Connect(url string, ctx context.Context, opts ...Option) (<-chan []byte, error) {
	c := &connection{
		outChan:   make(chan []byte),
		ctx:       ctx,
		header:    make(http.Header),
		reconnect: make(chan bool),
		dialer: websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		},
		writeWait:    15 * time.Second,
		pongWait:     30 * time.Second,
		pingInterval: (30 * time.Second * 9) / 10,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c.connect(url)
}

// Option функция настройки соединения.
type Option func(*connection)

// WithHandshake устанавливает данные для рукопожатия.
func WithHandshake(h []byte) Option {
	return func(c *connection) { c.handshake = h }
}

// WithHeader добавляет HTTP заголовки.
func WithHeader(h http.Header) Option {
	return func(c *connection) { c.header = h }
}

// WithWriteTimeout устанавливает таймаут записи.
func WithWriteTimeout(d time.Duration) Option {
	return func(c *connection) { c.writeWait = d }
}

// WithPongTimeout устанавливает таймаут pong и интервал ping (90% от pong).
func WithPongTimeout(d time.Duration) Option {
	return func(c *connection) {
		if d > 5*time.Second {
			c.pongWait = d
			c.pingInterval = time.Duration(float64(d) * 0.9)
		}
	}
}

// connect устанавливает соединение и возвращает канал для чтения.
func (c *connection) connect(url string) (<-chan []byte, error) {
	conn, _, err := c.dialer.Dial(url, c.header)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}
	c.conn = conn
	if len(c.handshake) > 0 {
		if err := c.writeMessage(websocket.TextMessage, c.handshake); err != nil {
			conn.Close()
			return nil, fmt.Errorf("ошибка рукопожатия: %w", err)
		}
	}
	go c.runPumps(url)
	return c.outChan, nil
}

// runPumps запускает обработку входящих/исходящих сообщений.
func (c *connection) runPumps(url string) {
	c.wg.Add(2)
	go c.readPump()
	go c.writePump()
	c.wg.Wait()

	c.conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			close(c.outChan)
			close(c.reconnect)
			return
		case <-ticker.C:
			if _, err := c.connect(url); err == nil {
				return
			}
		}
	}
}

// readPump обрабатывает входящие сообщения.
func (c *connection) readPump() {
	defer c.signalReconnect()

	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		select {
		case <-c.ctx.Done():
			return
		case c.outChan <- msg:
		}
	}
}

// writePump отправляет ping-сообщения.
func (c *connection) writePump() {
	defer c.signalReconnect()

	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.writeMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// signalReconnect уведомляет о переподключении.
func (c *connection) signalReconnect() {
	c.wg.Done()
	select {
	case c.reconnect <- true:
	default:
	}
}

// writeMessage отправляет сообщение с таймаутом.
func (c *connection) writeMessage(msgType int, data []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.conn.WriteMessage(msgType, data)
}
