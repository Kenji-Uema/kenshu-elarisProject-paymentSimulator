package db

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Kenji-Uema/paymentSimulator/internal/config"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	mongoC            *mongodb.MongoDBContainer
	invoiceRepository *invoiceRepo
	receiptRepository *receiptRepo
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	mongoC, err = runMongoContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start mongo container: %v", err)
	}

	uri, err := mongoC.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get mongo connection string: %v", err)
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		log.Fatalf("failed to parse connection string: %v", err)
	}
	mongoHost := parsedURI.Host
	if parsedURI.RawQuery != "" {
		mongoHost = fmt.Sprintf("%s/?%s", parsedURI.Host, parsedURI.RawQuery)
	}

	db, err := connectMongoWithRetry(context.Background(), config.MongoConfig{
		Username:                        "test_user",
		Password:                        "test_pass",
		Host:                            mongoHost,
		Database:                        "test_db",
		ConnectionTimeoutInSeconds:      10,
		ServerSelectionTimeoutInSeconds: 10,
		PingTimeoutInSeconds:            5,
		StartupTimeoutInSeconds:         20,
		MaxConnIdleTimeInSeconds:        60,
		MaxPoolSize:                     100,
		MinPoolSize:                     0,
		RetryWrites:                     true,
	}, 45*time.Second, 1*time.Second)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}

	invoiceRepository = &invoiceRepo{collection: db.Database.Collection("invoice")}
	receiptRepository = &receiptRepo{collection: db.Database.Collection("receipt")}

	code := m.Run()
	db.Close(context.Background())
	_ = testcontainers.TerminateContainer(mongoC)

	os.Exit(code)
}

func connectMongoWithRetry(ctx context.Context, cfg config.MongoConfig, timeout time.Duration, interval time.Duration) (*Db, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		db, err := NewMongoDb(ctx, cfg)
		if err == nil {
			return db, nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect mongo with retry timed out after %s: %w", timeout, lastErr)
		}

		time.Sleep(interval)
	}
}

func runMongoContainer(ctx context.Context) (container *mongodb.MongoDBContainer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to start MongoDB container: %v", r)
		}
	}()

	container, err = mongodb.Run(
		ctx,
		"mongo:latest",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_pass"),
	)
	return container, err
}

func setupAndRun(testName string, t *testing.T, test func(t *testing.T, ct *mongo.Collection, br *mongo.Collection)) {
	invoiceCollection := invoiceRepository.collection
	receiptCollection := receiptRepository.collection

	seed(t, invoiceCollection, "../../../test_data/invoices_fixture.json")
	seed(t, receiptCollection, "../../../test_data/receipts_fixture.json")

	t.Cleanup(func() {
		_ = invoiceCollection.Drop(context.Background())
		_ = receiptCollection.Drop(context.Background())
	})

	t.Run(testName, func(t *testing.T) {
		test(t, invoiceCollection, receiptCollection)
	})
}

func seed(t *testing.T, collection *mongo.Collection, filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	var rawDocs []bson.M
	if err := bson.UnmarshalExtJSON(data, false, &rawDocs); err != nil {
		t.Fatal(err)
	}

	docs := make([]interface{}, 0, len(rawDocs))
	for _, doc := range rawDocs {
		delete(doc, "_invalid")
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return
	}

	if _, err := collection.InsertMany(context.Background(), docs); err != nil {
		t.Fatal(err)
	}
}
