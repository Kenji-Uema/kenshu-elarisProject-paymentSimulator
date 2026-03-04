package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitDialAttempts = 5
	rabbitDialBackoff  = 2 * time.Second
)

type RabbitMqConnection struct {
	mu     sync.RWMutex
	conn   *amqp.Connection
	cfg    config.RabbitMqConfig
	closed bool
}

func NewRabbitMqConnection(ctx context.Context, cfg config.RabbitMqConfig) (*RabbitMqConnection, error) {
	c := &RabbitMqConnection{cfg: cfg}
	if err := c.reconnectLocked(); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "rabbitmq connection established")

	return c, nil
}

func (c *RabbitMqConnection) IsConnectionOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return false
	}

	return !c.conn.IsClosed()
}

func (c *RabbitMqConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.conn == nil || c.conn.IsClosed() {
		return nil
	}

	return c.conn.Close()
}

func (c *RabbitMqConnection) Channel() (*amqp.Channel, error) {
	conn, err := c.openConnection()
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil && conn.IsClosed() {
		conn, openErr := c.openConnection()
		if openErr != nil {
			return nil, openErr
		}
		return conn.Channel()
	}

	return ch, err
}

func (c *RabbitMqConnection) openConnection() (*amqp.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("rabbitmq connection closed")
	}

	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn, nil
	}

	if err := c.reconnectLocked(); err != nil {
		return nil, err
	}

	return c.conn, nil
}

func (c *RabbitMqConnection) reconnectLocked() error {
	if c.closed {
		return errors.New("rabbitmq connection closed")
	}

	uri := amqp.URI{
		Scheme:   "amqp",
		Username: string(c.cfg.Username),
		Password: string(c.cfg.Password),
		Host:     c.cfg.Host,
		Port:     c.cfg.Port,
	}

	var lastErr error
	for attempt := 1; attempt <= rabbitDialAttempts; attempt++ {
		conn, err := amqp.Dial(uri.String())
		if err == nil {
			c.conn = conn
			return nil
		}

		lastErr = err
		slog.Warn("rabbitmq dial failed", "attempt", attempt, "max_attempts", rabbitDialAttempts, "error", err)
		if attempt < rabbitDialAttempts {
			time.Sleep(rabbitDialBackoff)
		}
	}

	return fmt.Errorf("dial rabbitmq after %d attempts: %w", rabbitDialAttempts, lastErr)
}
