package validation

import (
	"log/slog"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestValidator_NotNilObjectID(t *testing.T) {
	tests := []struct {
		name      string
		id        bson.ObjectID
		shouldErr bool
	}{
		{name: "nil object id", id: bson.NilObjectID, shouldErr: true},
		{name: "non nil object id", id: bson.NewObjectID(), shouldErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().NotNilObjectID("id", tt.id).Validate()
			slog.Info("returned err", "err", err)

			if tt.shouldErr && err == nil {
				t.Fatalf("expected validation error for %s, got nil", tt.id.Hex())
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("expected no error for %s, got %v", tt.id.Hex(), err)
			}
		})
	}
}

func FuzzValidator_NoDuplicates(f *testing.F) {
	f.Add([]byte{1, 2, 3}) // no duplicates
	f.Add([]byte{1, 2, 1}) // has duplicates
	f.Add([]byte{})        // empty slice

	f.Fuzz(func(t *testing.T, input []byte) {
		err := New().NoDuplicates("ids", input).Validate()
		slog.Info("returned err", "err", err)

		seen := make(map[byte]struct{}, len(input))
		hasDuplicates := false
		for _, v := range input {
			if _, ok := seen[v]; ok {
				hasDuplicates = true
				break
			}
			seen[v] = struct{}{}
		}

		if hasDuplicates && err == nil {
			t.Fatalf("expected validation error for input with duplicates: %#v", input)
		}
		if !hasDuplicates && err != nil {
			t.Fatalf("expected no error for input without duplicates: %#v, got %v", input, err)
		}
	})
}
