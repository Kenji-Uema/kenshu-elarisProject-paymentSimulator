package config

import amqp "github.com/rabbitmq/amqp091-go"

type ExchangeConfig struct {
	Name       string     `env:"NAME,required"`
	Kind       string     `env:"KIND,required"`
	Durable    bool       `env:"DURABLE"`
	AutoDelete bool       `env:"AUTO_DELETE"`
	Internal   bool       `env:"INTERNAL"`
	NoWait     bool       `env:"NO_WAIT"`
	Args       amqp.Table `env:"ARGS"`
}

type QueueConfig struct {
	Name       string     `env:"NAME,required"`
	Durable    bool       `env:"DURABLE"`
	AutoDelete bool       `env:"AUTO_DELETE"`
	Exclusive  bool       `env:"EXCLUSIVE"`
	NoWait     bool       `env:"NO_WAIT"`
	Args       amqp.Table `env:"ARGS"`
}

type BindingConfig struct {
	ExchangeName string     `env:"EXCHANGE_NAME"`
	RoutingKey   string     `env:"ROUTING_KEY"`
	NoWait       bool       `env:"NO_WAIT"`
	Args         amqp.Table `env:"ARGS"`
}

type PublishConfig struct {
	Mandatory bool `env:"MANDATORY"`
	Immediate bool `env:"IMMEDIATE"`
}

type ConsumeConfig struct {
	Consumer  string     `env:"CONSUMER"`
	AutoAck   bool       `env:"AUTO_ACK"`
	Exclusive bool       `env:"EXCLUSIVE"`
	NoLocal   bool       `env:"NO_LOCAL"`
	NoWait    bool       `env:"NO_WAIT"`
	Args      amqp.Table `env:"ARGS"`
}
