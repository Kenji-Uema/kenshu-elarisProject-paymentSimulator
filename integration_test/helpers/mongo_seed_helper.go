package helpers

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func SeedMongoFromFixtures(t TestReporter, mongoHost string, database string, invoicesFixturePath string, receiptsFixturePath string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://test_user:test_pass@%s", mongoHost)))
	if err != nil {
		t.Fatalf("connect mongo for fixture seed: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	db := client.Database(database)
	seedCollectionFromFixture(ctx, t, db.Collection("invoices"), invoicesFixturePath)
	seedCollectionFromFixture(ctx, t, db.Collection("receipts"), receiptsFixturePath)
}

func seedCollectionFromFixture(ctx context.Context, t TestReporter, collection *mongo.Collection, fixturePath string) {
	t.Helper()

	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixturePath, err)
	}

	var rawDocs []bson.M
	if err := bson.UnmarshalExtJSON(data, false, &rawDocs); err != nil {
		t.Fatalf("unmarshal fixture %q: %v", fixturePath, err)
	}

	docs := make([]any, 0, len(rawDocs))
	for _, doc := range rawDocs {
		delete(doc, "_invalid")
		docs = append(docs, doc)
	}

	if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatalf("clear collection %q before seeding: %v", collection.Name(), err)
	}
	if len(docs) == 0 {
		return
	}

	if _, err := collection.InsertMany(ctx, docs); err != nil {
		t.Fatalf("insert fixture docs into %q: %v", collection.Name(), err)
	}
}
