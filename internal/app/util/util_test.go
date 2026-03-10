package util

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateHumanFriendlyId(t *testing.T) {
	t.Run("should generate human friendly id", func(t *testing.T) {
		now := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
		got, err := GenerateHumanFriendlyId("test", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := strings.Split(got, "-")
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts separated by '-', got %q", got)
		}

		if parts[0] != "test" {
			t.Fatalf("expected prefix %q, got %q", "test", parts[0])
		}
		if parts[1] != "20260308" {
			t.Fatalf("expected date %q, got %q", "20260308", parts[1])
		}
		if len(parts[2]) != invoiceNumberSuffixLength {
			t.Fatalf("expected suffix len %d, got %d", invoiceNumberSuffixLength, len(parts[2]))
		}
	})
}
