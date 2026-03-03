package validation

import (
	"log/slog"
	"testing"
	"time"
)

func TestValidator_Period(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name      string
		start     time.Time
		end       time.Time
		shouldErr bool
	}{
		{
			name:      "valid period",
			start:     now,
			end:       now.Add(24 * time.Hour),
			shouldErr: false,
		},
		{
			name:      "equal boundaries",
			start:     now,
			end:       now,
			shouldErr: true,
		},
		{
			name:      "start after end",
			start:     now,
			end:       now.Add(-24 * time.Hour),
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().ValidPeriod(tt.start, tt.end).Validate()
			slog.Info("returned err", "err", err)

			if tt.shouldErr && err == nil {
				t.Fatalf("expected validation error for start=%v end=%v, got nil", tt.start, tt.end)
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("expected no error for start=%v end=%v, got %v", tt.start, tt.end, err)
			}
		})
	}
}
