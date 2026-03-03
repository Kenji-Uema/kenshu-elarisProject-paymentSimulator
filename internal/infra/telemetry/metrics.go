package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter                 = otel.Meter("clock-emulator")
	StartupHistogram      metric.Int64Histogram
	MqPublishCounter      metric.Int64Counter
	MqPublishDurationHist metric.Float64Histogram
)

func initMetrics() error {
	var err error
	StartupHistogram, err = meter.Int64Histogram(
		"app.startup.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Time from process start to ready state"),
	)
	if err != nil {
		return err
	}

	MqPublishCounter, err = meter.Int64Counter(
		"mq.publish",
		metric.WithDescription("Number of RabbitMQ publish attempts."),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	MqPublishDurationHist, err = meter.Float64Histogram(
		"mq.publish.duration_ms",
		metric.WithDescription("RabbitMQ publish end-to-end duration in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	return nil
}
