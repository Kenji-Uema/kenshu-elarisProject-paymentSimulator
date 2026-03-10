package mq

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/mqErrors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqChannel struct {
	*RabbitMqConnection
	channel *amqp.Channel
}

func NewRabbitMqChannel(rabbitmqConnection *RabbitMqConnection) *RabbitMqChannel {
	return &RabbitMqChannel{RabbitMqConnection: rabbitmqConnection}
}

func (r *RabbitMqChannel) openChannel() error {
	conn, err := r.openConnection()
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		// If the underlying connection died between checks, reopen and retry once.
		if !conn.IsClosed() {
			return err
		}

		conn, openErr := r.openConnection()
		if openErr != nil {
			return openErr
		}

		ch, err = conn.Channel()
		if err != nil {
			return err
		}
	}

	r.channel = ch
	return nil
}

func (r *RabbitMqChannel) reopenChannel(ctx context.Context) error {
	slog.Warn("channel is closed, opening a new one")

	if err := r.openChannel(); err != nil {
		slog.ErrorContext(ctx, "failed to open channel", "error", err)
		return &mqErrors.UnexpectedErr{Msg: "unexpected error when reopening channel", Err: err}
	}

	return nil
}

func (r *RabbitMqChannel) CloseChannel() error {
	if r.channel == nil || r.channel.IsClosed() {
		return nil
	}

	return r.channel.Close()
}
