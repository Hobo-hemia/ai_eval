//go:build m3harness

package result

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShardCache_NewConfigValidation_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     CacheConfig
		wantErr bool
	}{
		{
			name: "invalid shard count",
			cfg: CacheConfig{
				ShardCount:         0,
				MaxEntriesPerShard: 8,
				DefaultTTL:         2 * time.Second,
				CleanupInterval:    50 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid max entries",
			cfg: CacheConfig{
				ShardCount:         8,
				MaxEntriesPerShard: 0,
				DefaultTTL:         2 * time.Second,
				CleanupInterval:    50 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "valid config",
			cfg: CacheConfig{
				ShardCount:         8,
				MaxEntriesPerShard: 8,
				DefaultTTL:         200 * time.Millisecond,
				CleanupInterval:    20 * time.Millisecond,
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewShardCache[string, int](tc.cfg)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, c)
			assert.NoError(t, c.Close())
		})
	}
}

func TestShardCache_TTLAndEviction(t *testing.T) {
	t.Parallel()

	cache, err := NewShardCache[string, int](CacheConfig{
		ShardCount:         1,
		MaxEntriesPerShard: 2,
		DefaultTTL:         60 * time.Millisecond,
		CleanupInterval:    20 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	cache.Set(context.Background(), "a", 1, 0)
	v, ok := cache.Get(context.Background(), "a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	time.Sleep(90 * time.Millisecond)
	_, ok = cache.Get(context.Background(), "a")
	assert.False(t, ok, "entry should expire by default ttl")

	cache.Set(context.Background(), "k1", 1, time.Second)
	cache.Set(context.Background(), "k2", 2, time.Second)
	cache.Set(context.Background(), "k3", 3, time.Second)

	_, ok1 := cache.Get(context.Background(), "k1")
	_, ok2 := cache.Get(context.Background(), "k2")
	_, ok3 := cache.Get(context.Background(), "k3")
	assert.False(t, ok1, "oldest should be evicted when shard capacity exceeded")
	assert.True(t, ok2)
	assert.True(t, ok3)

	stats := cache.Stats()
	assert.GreaterOrEqual(t, stats.Evictions, uint64(1))
}

func TestShardCache_GetOrLoad_SameKeySingleflight(t *testing.T) {
	t.Parallel()

	cache, err := NewShardCache[string, int](CacheConfig{
		ShardCount:         16,
		MaxEntriesPerShard: 128,
		DefaultTTL:         time.Second,
		CleanupInterval:    100 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	var loaderCalls int64
	loader := func(ctx context.Context, key string) (int, time.Duration, error) {
		atomic.AddInt64(&loaderCalls, 1)
		time.Sleep(40 * time.Millisecond)
		return len(key), time.Second, nil
	}

	const workers = 64
	var wg sync.WaitGroup
	wg.Add(workers)
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			v, loadErr := cache.GetOrLoad(context.Background(), "single-key", loader)
			if v != len("single-key") {
				errCh <- errors.New("unexpected loaded value")
				return
			}
			errCh <- loadErr
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		assert.NoError(t, e)
	}
	assert.Equal(t, int64(1), atomic.LoadInt64(&loaderCalls), "same key should load only once")

	stats := cache.Stats()
	assert.GreaterOrEqual(t, stats.LoadSuccess, uint64(1))
}

func TestShardCache_GetOrLoad_SlowLoaderDoesNotBlockOtherKey(t *testing.T) {
	t.Parallel()

	cache, err := NewShardCache[string, int](CacheConfig{
		ShardCount:         8,
		MaxEntriesPerShard: 64,
		DefaultTTL:         time.Second,
		CleanupInterval:    100 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	release := make(chan struct{})
	started := make(chan string, 2)

	loader := func(ctx context.Context, key string) (int, time.Duration, error) {
		started <- key
		<-release
		return len(key), time.Second, nil
	}

	errCh := make(chan error, 2)
	go func() {
		_, e := cache.GetOrLoad(context.Background(), "alpha", loader)
		errCh <- e
	}()

	select {
	case <-started:
	case <-time.After(120 * time.Millisecond):
		t.Fatalf("first loader did not start in time")
	}

	go func() {
		_, e := cache.GetOrLoad(context.Background(), "beta", loader)
		errCh <- e
	}()

	select {
	case <-started:
		// second key loader should start before first is released.
	case <-time.After(120 * time.Millisecond):
		t.Fatalf("second key blocked by lock scope, expected non-blocking behavior")
	}

	close(release)
	assert.NoError(t, <-errCh)
	assert.NoError(t, <-errCh)
}

func TestShardCache_GetOrLoad_ContextCanceledFastFail(t *testing.T) {
	t.Parallel()

	cache, err := NewShardCache[string, int](CacheConfig{
		ShardCount:         4,
		MaxEntriesPerShard: 32,
		DefaultTTL:         time.Second,
		CleanupInterval:    50 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer cache.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := int64(0)
	loader := func(ctx context.Context, key string) (int, time.Duration, error) {
		atomic.AddInt64(&called, 1)
		return 1, time.Second, nil
	}
	_, err = cache.GetOrLoad(ctx, "cancelled", loader)
	assert.Error(t, err)
	assert.Equal(t, int64(0), atomic.LoadInt64(&called), "cancelled context should not invoke loader")
}

func BenchmarkShardCacheGetHitParallel(b *testing.B) {
	cache, err := NewShardCache[int, int](CacheConfig{
		ShardCount:         32,
		MaxEntriesPerShard: 512,
		DefaultTTL:         time.Second,
		CleanupInterval:    time.Second,
	})
	if err != nil {
		b.Fatalf("new cache failed: %v", err)
	}
	defer cache.Close()

	for i := 0; i < 4096; i++ {
		cache.Set(context.Background(), i, i*10, time.Second)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := 0
		for pb.Next() {
			cache.Get(context.Background(), key%4096)
			key++
		}
	})
}
