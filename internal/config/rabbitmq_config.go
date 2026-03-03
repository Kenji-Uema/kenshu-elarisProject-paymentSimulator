package config

import amqp "github.com/rabbitmq/amqp091-go"

type ExchangeConfig struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

type BindingConfig struct {
	ExchangeName string
	RoutingKey   string
	NoWait       bool
	Args         amqp.Table
}

type PublishConfig struct {
	RoutingKey string
	Mandatory  bool
	Immediate  bool
}

type ConsumeConfig struct {
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}
