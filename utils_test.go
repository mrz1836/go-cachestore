package cachestore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRandomHex will test the method RandomHex()
func TestRandomHex(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name           string
		input          int
		expectedLength int
	}{
		{"zero", 0, 0},
		{"one", 1, 2},
		{"100k", 100000, 200000},
		{"16->32", 16, 32},
		{"32->64", 32, 64},
		{"8->16", 8, 16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := RandomHex(test.input)
			require.NoError(t, err)
			assert.Len(t, output, test.expectedLength)
		})
	}

	t.Run("panic with a negative", func(t *testing.T) {
		assert.Panics(t, func() {
			_, err := RandomHex(-1)
			require.Error(t, err)
		})
	})

	t.Run("log an example", func(t *testing.T) {
		output, err := RandomHex(32)
		require.NoError(t, err)
		t.Log(output)
	})
}
