package mq

import (
	"context"
	"errors"
	"testing"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"github.com/Kenji-Uema/paymentSimulator/internal/domain/errors/validationErrors"
)

func TestNewRabbitMqConnection_InvalidConfig(t *testing.T) {
	_, err := NewRabbitMqConnection(context.Background(), config.RabbitMqConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}

func TestNewRabbitMqChannel_InvalidConnection(t *testing.T) {
	_, err := NewRabbitMqChannel(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}

func TestNewRabbitmqProducer_InvalidConnection(t *testing.T) {
	_, err := NewRabbitmqProducer(nil, config.PublishConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}

func TestNewRabbitmqConsumer_InvalidConnection(t *testing.T) {
	_, err := NewRabbitmqConsumer(nil, config.ConsumeConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var validationErr *validationErrors.ErrValidationConstrain
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validationErrors.ErrValidationConstrain, got %T", err)
	}
}
