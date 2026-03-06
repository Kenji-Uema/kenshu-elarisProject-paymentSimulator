package util

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const invoiceNumberSuffixLength = 6

func GenerateHumanFriendlyId(prefix string, now time.Time) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	source := strings.ReplaceAll(id.String(), "-", "")

	random := make([]byte, invoiceNumberSuffixLength)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	suffix := make([]byte, invoiceNumberSuffixLength)
	for i := 0; i < invoiceNumberSuffixLength; i++ {
		suffix[i] = source[int(random[i])%len(source)]
	}

	return fmt.Sprintf("%s-%s-%s", prefix, now.UTC().Format("20060102"), string(suffix)), nil
}
