package backoff_test

import (
	"testing"

	"github.com/bhoriuchi/graphql-go-server/utils/backoff"
)

func TestBackoff(t *testing.T) {
	b := backoff.NewBackoff(&backoff.Options{
		Jitter: 0.5,
	})
	for i := 1; i < 11; i++ {
		dur := b.Duration()
		t.Logf("Loop %d: %d\n", i, dur.Milliseconds())
	}
}
