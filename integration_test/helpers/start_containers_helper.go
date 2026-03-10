package helpers

import (
	"context"
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func StartContainers(ctx context.Context) (mongoHost string, rabbitHost string, rabbitPort int) {
	mongoC, mongoHost, err := startMongoContainer(ctx)
	if err != nil {
		Skip(fmt.Sprintf("skipping integration test: failed to start mongo container: %v", err))
	}
	DeferCleanup(func() { _ = testcontainers.TerminateContainer(mongoC) })

	rabbitC, rabbitHost, rabbitPort, err := startRabbitContainer(ctx)
	if err != nil {
		Skip(fmt.Sprintf("skipping integration test: failed to start rabbitmq container: %v", err))
	}
	DeferCleanup(func() { _ = testcontainers.TerminateContainer(rabbitC) })

	return
}

func startMongoContainer(ctx context.Context) (container *mongodb.MongoDBContainer, host string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to start mongo container: %v", r)
		}
	}()

	container, err = mongodb.Run(
		ctx,
		"mongo:latest",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_pass"),
	)
	if err != nil {
		return nil, "", err
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, "", err
	}
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	host = parsedURI.Host
	if parsedURI.RawQuery != "" {
		host = fmt.Sprintf("%s/?%s", parsedURI.Host, parsedURI.RawQuery)
	}

	return container, host, nil
}

func startRabbitContainer(ctx context.Context) (container *rabbitmq.RabbitMQContainer, host string, port int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to start rabbitmq container: %v", r)
		}
	}()

	rabbitContainer, err := rabbitmq.Run(
		ctx,
		"rabbitmq:3.13",
		rabbitmq.WithAdminUsername("test_user"),
		rabbitmq.WithAdminPassword("test_pass"),
	)
	if err != nil {
		return nil, "", 0, err
	}
	container = rabbitContainer

	host, err = container.Host(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	mappedPort, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		return nil, "", 0, err
	}

	return container, host, mappedPort.Int(), nil
}
