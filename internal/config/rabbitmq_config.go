package config

type ExchangeConfig struct {
	Name       string `env:"NAME,required"`
	Kind       string `env:"KIND,required"`
	Durable    bool   `env:"DURABLE" envDefault:"true"`
	AutoDelete bool   `env:"AUTO_DELETE" envDefault:"false"`
	Internal   bool   `env:"INTERNAL" envDefault:"false"`
	NoWait     bool   `env:"NO_WAIT" envDefault:"false"`
}

type QueueConfig struct {
	Name       string `env:"NAME,required"`
	Durable    bool   `env:"DURABLE" envDefault:"true"`
	AutoDelete bool   `env:"AUTO_DELETE" envDefault:"false"`
	Exclusive  bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoWait     bool   `env:"NO_WAIT" envDefault:"false"`
}

type BindingConfig struct {
	ExchangeName string `env:"EXCHANGE_NAME"`
	RoutingKey   string `env:"ROUTING_KEY"`
	NoWait       bool   `env:"NO_WAIT" envDefault:"false"`
}

type PublishConfig struct {
	Mandatory bool `env:"MANDATORY" envDefault:"false"`
	Immediate bool `env:"IMMEDIATE" envDefault:"false"`
}

type ConsumeConfig struct {
	Consumer  string `env:"CONSUMER"`
	AutoAck   bool   `env:"AUTO_ACK" envDefault:"false"`
	Exclusive bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoLocal   bool   `env:"NO_LOCAL" envDefault:"false"`
	NoWait    bool   `env:"NO_WAIT" envDefault:"false"`
}
