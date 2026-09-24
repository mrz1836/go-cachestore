package cachestore

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetModelValue tests the generic GetModelValue helper across both engines.
func TestGetModelValue(t *testing.T) {
	for _, testCase := range getInMemoryTestCases(t) {
		t.Run(testCase.name+" - round trip returns the decoded value", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)
			defer func() { _ = c.EmptyCache(context.Background()) }()

			want := genericStruct{BoolField: true, FloatField: 123.123, IntField: 7, StringField: "hello"}
			require.NoError(t, SetModelValue(ctx, c, testKey, want, time.Minute))

			got, found, gErr := GetModelValue[genericStruct](ctx, c, testKey)
			require.NoError(t, gErr)
			assert.True(t, found)
			assert.Equal(t, want, got)
		})

		t.Run(testCase.name+" - miss returns the zero value and not found", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)
			defer func() { _ = c.EmptyCache(context.Background()) }()

			got, found, gErr := GetModelValue[genericStruct](ctx, c, testKey+"-missing")
			require.NoError(t, gErr)
			assert.False(t, found)
			assert.Equal(t, genericStruct{}, got)
		})

		t.Run(testCase.name+" - stored value that is not valid JSON errors", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)
			defer func() { _ = c.EmptyCache(context.Background()) }()

			require.NoError(t, c.Set(ctx, testKey, "not-json"))

			got, found, gErr := GetModelValue[genericStruct](ctx, c, testKey)
			require.Error(t, gErr)
			assert.False(t, found)
			assert.Equal(t, genericStruct{}, got)
		})

		t.Run(testCase.name+" - empty key errors", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)

			_, found, gErr := GetModelValue[genericStruct](ctx, c, "")
			require.Error(t, gErr)
			require.ErrorIs(t, gErr, ErrKeyRequired)
			assert.False(t, found)
		})
	}
}

// TestSetModelValue tests the generic SetModelValue helper across both engines.
func TestSetModelValue(t *testing.T) {
	for _, testCase := range getInMemoryTestCases(t) {
		t.Run(testCase.name+" - stores a value readable by GetModel", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)
			defer func() { _ = c.EmptyCache(context.Background()) }()

			want := genericStruct{IntField: 42, StringField: "world"}
			require.NoError(t, SetModelValue(ctx, c, testKey, want, time.Minute))

			got := new(genericStruct)
			require.NoError(t, c.GetModel(ctx, testKey, got))
			assert.Equal(t, 42, got.IntField)
			assert.Equal(t, "world", got.StringField)
		})

		t.Run(testCase.name+" - empty key errors", func(t *testing.T) {
			ctx := context.Background()
			c, err := NewClient(ctx, testCase.opts)
			require.NoError(t, err)
			require.NotNil(t, c)

			err = SetModelValue(ctx, c, "", genericStruct{}, time.Minute)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrKeyRequired)
		})
	}
}
