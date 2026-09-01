package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCacheGetSet(t *testing.T) {
	cfg := Config{
		Type:          "memory",
		DefaultTTL:    1 * time.Second,
		FailedTTL:     2 * time.Second,
		MaxMemorySize: 100,
	}

	c := NewMemoryCache(cfg)

	ctx := context.Background()
	key := GenerateCacheKey("token-123-boletim-read")

	val := &ValidationResult{
		Success:   true,
		Action:    "boletim:read",
		Hostname:  "boletim.apps.rio.gov.br",
		Timestamp: time.Now(),
	}
	err := c.Set(ctx, key, val, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	retrieved, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get cache: %v", err)
	}
	if retrieved == nil || !retrieved.Success || retrieved.Action != "boletim:read" {
		t.Errorf("expected successful retrieval with action boletim:read, got %+v", retrieved)
	}

	time.Sleep(600 * time.Millisecond)
	_, err = c.Get(ctx, key)
	if err == nil {
		t.Errorf("expected cache key to expire, but still found")
	}
}

func TestGenerateCacheKeyDeterminism(t *testing.T) {
	k1 := GenerateCacheKey("token1")
	k2 := GenerateCacheKey("token1")
	k3 := GenerateCacheKey("token2")

	if k1 != k2 {
		t.Errorf("expected identical keys for same input, got %s vs %s", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("expected different keys for different tokens, got matching %s", k1)
	}
}

func TestSplitSentinelHostsTrimsAndDropsEmpty(t *testing.T) {
	got := splitSentinelHosts(" sentinel-1:26379 ,sentinel-2:26379,, sentinel-3:26379")
	want := []string{"sentinel-1:26379", "sentinel-2:26379", "sentinel-3:26379"}

	if len(got) != len(want) {
		t.Fatalf("expected %d hosts, got %d (%v)", len(want), len(got), got)
	}
	for i, h := range want {
		if got[i] != h {
			t.Errorf("host %d: expected %q, got %q", i, h, got[i])
		}
	}
}

func TestSplitSentinelHostsEmptyInput(t *testing.T) {
	if got := splitSentinelHosts(""); got != nil {
		t.Errorf("expected nil for empty input, got %v", got)
	}
}

func TestNewRedisClientStandaloneURLFallback(t *testing.T) {
	cfg := Config{
		RedisURL: "redis://:secret@localhost:6380/3",
	}

	client, err := newRedisClient(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	opts := client.Options()
	if opts.Addr != "localhost:6380" {
		t.Errorf("expected addr localhost:6380, got %s", opts.Addr)
	}
	if opts.Password != "secret" {
		t.Errorf("expected password 'secret', got %q", opts.Password)
	}
	if opts.DB != 3 {
		t.Errorf("expected db 3, got %d", opts.DB)
	}
}

func TestNewRedisClientInvalidURLReturnsError(t *testing.T) {
	cfg := Config{RedisURL: "not-a-valid-redis-url"}

	if _, err := newRedisClient(cfg); err == nil {
		t.Error("expected error for invalid Redis URL, got nil")
	}
}

func TestNewRedisClientSentinelTakesPrecedenceOverURL(t *testing.T) {
	cfg := Config{
		// Deliberately invalid so a fallback-to-URL path would fail this test.
		RedisURL:             "not-a-valid-redis-url",
		RedisSentinelHosts:   " sentinel-1:26379 , sentinel-2:26379 ",
		RedisSentinelService: "mymaster",
		RedisPassword:        "sentinel-secret",
	}

	client, err := newRedisClient(cfg)
	if err != nil {
		t.Fatalf("expected sentinel config to be used without touching RedisURL, got error: %v", err)
	}

	opts := client.Options()
	if opts.Password != "sentinel-secret" {
		t.Errorf("expected password 'sentinel-secret', got %q", opts.Password)
	}
	if opts.DB != 0 {
		t.Errorf("expected default db 0, got %d", opts.DB)
	}
}

func TestNewRedisClientSentinelRequiresBothHostsAndService(t *testing.T) {
	// Only hosts set, no service name: must fall back to standalone URL parsing.
	cfg := Config{
		RedisURL:           "redis://localhost:6379/0",
		RedisSentinelHosts: "sentinel-1:26379",
	}

	client, err := newRedisClient(cfg)
	if err != nil {
		t.Fatalf("expected fallback to standalone URL, got error: %v", err)
	}
	if client.Options().Addr != "localhost:6379" {
		t.Errorf("expected fallback to standalone addr, got %s", client.Options().Addr)
	}
}

func TestNewCacheFallsBackToMemoryWithoutRedisTarget(t *testing.T) {
	cfg := Config{
		Type:          "redis",
		MaxMemorySize: 10,
	}

	c, err := NewCache(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := c.(*memoryCache); !ok {
		t.Errorf("expected memory cache fallback when no Redis URL or Sentinel config is set, got %T", c)
	}
}
