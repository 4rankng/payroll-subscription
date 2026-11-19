package asynq

import (
	"testing"

	"api-server/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestEnqueueDuplicateJobsRemainIdempotent(t *testing.T) {
	redis := miniredis.RunT(t)
	client, err := NewClient(config.AsynqConfig{RedisAddr: redis.Addr()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.EnqueueBCCImport(123))
	require.NoError(t, client.EnqueueBCCImport(123))
}
