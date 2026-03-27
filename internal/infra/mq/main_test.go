package mq

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/port"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

var (
	rabbitContainer *rabbitmq.RabbitMQContainer
	rabbitConn      *RabbitMqConnection
)

func TestMain(m *testing.M) {
	cleanup, err := prepareTestcontainersTempDir()
	if err != nil {
		panic(fmt.Sprintf("failed to prepare temp dir: %v", err))
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	container, cfg, err := runRabbitMQContainer(ctx)
	if err != nil {
		panic(fmt.Sprintf("failed to start rabbitmq container: %v", err))
	}
	rabbitContainer = container

	rabbitConn, err = NewRabbitMqConnection(context.Background(), cfg)
	if err != nil {
		_ = testcontainers.TerminateContainer(rabbitContainer)
		panic(fmt.Sprintf("failed to create rabbitmq connection: %v", err))
	}

	code := m.Run()

	if rabbitConn != nil {
		_ = rabbitConn.Close()
	}
	if rabbitContainer != nil {
		_ = testcontainers.TerminateContainer(rabbitContainer)
	}

	os.Exit(code)
}

func prepareTestcontainersTempDir() (func(), error) {
	tempDir, err := os.MkdirTemp("", "testcontainers-tmp-")
	if err != nil {
		return nil, err
	}

	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		if err := os.Setenv(key, tempDir); err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, err
		}
	}

	return func() {
		_ = os.RemoveAll(tempDir)
	}, nil
}

func runRabbitMQContainer(ctx context.Context) (container *rabbitmq.RabbitMQContainer, cfg config.RabbitMqConfig, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to start RabbitMQ container: %v", r)
		}
	}()

	container, err = rabbitmq.Run(
		ctx,
		"rabbitmq:3.13-management",
		rabbitmq.WithAdminUsername("test_user"),
		rabbitmq.WithAdminPassword("test_pass"),
	)
	if err != nil {
		return nil, config.RabbitMqConfig{}, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, config.RabbitMqConfig{}, err
	}

	mappedPort, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		return nil, config.RabbitMqConfig{}, err
	}

	cfg = config.RabbitMqConfig{
		Username: "test_user",
		Password: "test_pass",
		Host:     host,
		Port:     mappedPort.Int(),
	}

	return container, cfg, nil
}

func setupAndRun(testName string, t *testing.T,
	testFn func(t *testing.T, consumer port.MqConsumer, producer port.MqProducer)) {

	consumer, err := NewRabbitmqConsumer(rabbitConn, config.ConsumeConfig{})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}

	producer, err := NewRabbitmqProducer(rabbitConn, config.PublishConfig{})
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}

	t.Cleanup(func() {
		_ = consumer.CloseChannel()
		_ = producer.CloseChannel()
	})

	t.Run(testName, func(t *testing.T) {
		testFn(t, consumer, producer)
	})
}
