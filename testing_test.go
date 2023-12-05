package cachestore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetNewRelicApp(t *testing.T) {
	t.Parallel()

	t.Run("no app name", func(t *testing.T) {
		app, err := getNewRelicApp("")
		require.Error(t, err)
		require.Nil(t, app)
		require.ErrorIs(t, err, ErrAppNameRequired)
	})

	t.Run("valid app name", func(t *testing.T) {
		app, err := getNewRelicApp("test-app")
		require.NoError(t, err)
		require.NotNil(t, app)
	})
}
