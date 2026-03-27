package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/app/validation"
	"github.com/Kenji-Uema/paymentSimulator/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

type Db struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDb(ctx context.Context, cfg config.MongoConfig) (*Db, error) {
	if err := validation.New().
		NotBlank("mongo.username", string(cfg.Username)).
		NotBlank("mongo.password", string(cfg.Password)).
		NotBlank("mongo.host", cfg.Host).
		NotBlank("mongo.database", cfg.Database).
		PositiveValue("mongo.connection_timeout_in_seconds", cfg.ConnectionTimeoutInSeconds).
		PositiveValue("mongo.server_selection_timeout_in_seconds", cfg.ServerSelectionTimeoutInSeconds).
		PositiveValue("mongo.ping_timeout_in_seconds", cfg.PingTimeoutInSeconds).
		PositiveValue("mongo.startup_timeout_in_seconds", cfg.StartupTimeoutInSeconds).
		PositiveValue("mongo.max_conn_idle_time_in_seconds", cfg.MaxConnIdleTimeInSeconds).
		PositiveValue("mongo.max_pool_size", cfg.MaxPoolSize).
		Validate(); err != nil {
		return nil, err
	}

	startCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.StartupTimeoutInSeconds)*time.Second)
	defer cancel()

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s", string(cfg.Username), string(cfg.Password), cfg.Host)
	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetConnectTimeout(time.Duration(cfg.ConnectionTimeoutInSeconds) * time.Second).
		SetServerSelectionTimeout(time.Duration(cfg.ServerSelectionTimeoutInSeconds) * time.Second).
		SetMaxConnIdleTime(time.Duration(cfg.MaxConnIdleTimeInSeconds) * time.Second).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetRetryWrites(cfg.RetryWrites).
		SetMonitor(otelmongo.NewMonitor(
			otelmongo.WithCommandAttributeDisabled(true),
		))

	if cfg.ReplicaSet != "" {
		clientOptions.SetReplicaSet(cfg.ReplicaSet)
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(startCtx, time.Duration(cfg.PingTimeoutInSeconds)*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping failed for URI: %s, error: %w", mongoURI, err)
	}

	mongoDb := &Db{Client: client, Database: client.Database(cfg.Database)}
	slog.InfoContext(startCtx, "connected to MongoDB")
	return mongoDb, nil
}

func (d *Db) Close(ctx context.Context) error {
	return d.Client.Disconnect(ctx)
}

func (d *Db) Collection(name string) *mongo.Collection {
	return d.Database.Collection(name)
}

func (d *Db) Ping() error {
	return d.Client.Ping(context.Background(), readpref.Primary())
}
