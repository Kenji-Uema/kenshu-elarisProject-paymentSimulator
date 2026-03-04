package port

import (
	"context"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type MqProducer interface {
	DeclareExchange(config config.ExchangeConfig) error
	Publish(ctx context.Context, message proto.Message, routingKey string) error
	CloseChannel() error
}

type MqConsumer interface {
	DeclareQueue(config config.QueueConfig) error
	BindQueue(config config.BindingConfig) error
	Consume(ctx context.Context) (<-chan amqp.Delivery, error)
	CloseChannel() error
}
