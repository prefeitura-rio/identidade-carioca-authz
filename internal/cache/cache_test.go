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
