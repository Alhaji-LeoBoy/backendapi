package cache

import (
	"context"
	"testing"
	"time"
)

func TestDeleteByPattern_WithWildcardPrefix(t *testing.T) {
	t.Parallel()

	c := NewInMemoryCache()
	ctx := context.Background()

	_ = c.Set(ctx, "events:limit=10:offset=0", []byte(`{"ok":true}`), time.Minute)
	_ = c.Set(ctx, "events:search:go:limit=10:offset=0", []byte(`{"ok":true}`), time.Minute)
	_ = c.Set(ctx, "tickets:limit=10:offset=0", []byte(`{"ok":true}`), time.Minute)

	if err := c.DeleteByPattern(ctx, "events:*"); err != nil {
		t.Fatalf("DeleteByPattern returned error: %v", err)
	}

	if _, err := c.Get(ctx, "events:limit=10:offset=0"); err == nil {
		t.Fatalf("expected events list cache key to be deleted")
	}
	if _, err := c.Get(ctx, "events:search:go:limit=10:offset=0"); err == nil {
		t.Fatalf("expected events search cache key to be deleted")
	}
	if _, err := c.Get(ctx, "tickets:limit=10:offset=0"); err != nil {
		t.Fatalf("expected non-matching key to remain, got error: %v", err)
	}
}

func TestDeleteByPattern_WithPlainPrefix(t *testing.T) {
	t.Parallel()

	c := NewInMemoryCache()
	ctx := context.Background()

	_ = c.Set(ctx, "tickets:limit=10:offset=0", []byte(`{"ok":true}`), time.Minute)
	_ = c.Set(ctx, "ticket:42", []byte(`{"ok":true}`), time.Minute)

	if err := c.DeleteByPattern(ctx, "tickets:"); err != nil {
		t.Fatalf("DeleteByPattern returned error: %v", err)
	}

	if _, err := c.Get(ctx, "tickets:limit=10:offset=0"); err == nil {
		t.Fatalf("expected tickets prefix key to be deleted")
	}
	if _, err := c.Get(ctx, "ticket:42"); err != nil {
		t.Fatalf("expected near-match key to remain, got error: %v", err)
	}
}

