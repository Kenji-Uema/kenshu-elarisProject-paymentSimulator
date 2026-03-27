package helpers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	amqp "github.com/rabbitmq/amqp091-go"
)

func SetupRabbitmqChannel(t FullGinkgoTInterface, rabbitHost string, rabbitPort int) (amqpCh *amqp.Channel) {
	amqpConn, amqpCh := openAMQPChannel(t, rabbitHost, rabbitPort)
	DeferCleanup(func() {
		_ = amqpCh.Close()
		_ = amqpConn.Close()
	})

	return
}

func SetupRabbitmqExchangesAndQueues(t FullGinkgoTInterface, amqpCh *amqp.Channel) (paymentRequestDeliveries <-chan amqp.Delivery, confirmationDeliveries <-chan amqp.Delivery) {
	paymentRequestQueue := fmt.Sprintf("it.payment.request.%d", time.Now().UnixNano())
	confirmationQueue := fmt.Sprintf("it.payment.confirmation.%d", time.Now().UnixNano())

	MustDeclareExchange(t, amqpCh, "payment.events")
	MustDeclareAndBindQueue(t, amqpCh, paymentRequestQueue, "payment.events", "payment.payer-123.request")
	MustDeclareAndBindQueue(t, amqpCh, confirmationQueue, "payment.events", "booking.booking-123.confirmation")

	paymentRequestDeliveries, err := amqpCh.Consume(paymentRequestQueue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume payment request queue: %v", err)
	}
	confirmationDeliveries, err = amqpCh.Consume(confirmationQueue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume confirmation queue: %v", err)
	}

	return
}

func MustDeclareExchange(t TestReporter, ch *amqp.Channel, name string) {
	t.Helper()
	if err := ch.ExchangeDeclare(name, "direct", true, false, false, false, nil); err != nil {
		t.Fatalf("declare exchange %q: %v", name, err)
	}
}

func MustDeclareAndBindQueue(t TestReporter, ch *amqp.Channel, queueName string, exchangeName string, routingKey string) {
	t.Helper()
	if _, err := ch.QueueDeclare(queueName, false, true, false, false, nil); err != nil {
		t.Fatalf("declare queue %q: %v", queueName, err)
	}
	if err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		t.Fatalf("bind queue %q to exchange %q with routing key %q: %v", queueName, exchangeName, routingKey, err)
	}
}

func WaitForDelivery(t TestReporter, deliveries <-chan amqp.Delivery, timeout time.Duration) amqp.Delivery {
	t.Helper()
	select {
	case d, ok := <-deliveries:
		if !ok {
			t.Fatal("deliveries channel closed")
		}
		return d
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for delivery after %v", timeout)
		return amqp.Delivery{}
	}
}

func openAMQPChannel(t TestReporter, host string, port int) (*amqp.Connection, *amqp.Channel) {
	t.Helper()

	conn, err := amqp.Dial(fmt.Sprintf("amqp://test_user:test_pass@%s:%d/", host, port))
	if err != nil {
		t.Fatalf("open amqp connection: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("open amqp channel: %v", err)
	}

	return conn, ch
}
