package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/pyama86/slack-afk/go/store"
)

type fakeDatastore struct {
	kv  map[string]string
	ttl map[string]time.Duration
}

func newFakeDatastore() *fakeDatastore {
	return &fakeDatastore{
		kv:  make(map[string]string),
		ttl: make(map[string]time.Duration),
	}
}

func (f *fakeDatastore) Set(key string, value string) error {
	f.kv[key] = value
	return nil
}

func (f *fakeDatastore) SetEX(key string, value string, duration time.Duration) error {
	f.kv[key] = value
	f.ttl[key] = duration
	return nil
}

func (f *fakeDatastore) Get(key string) (string, error) {
	v, ok := f.kv[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeDatastore) Expire(key string, duration time.Duration) error {
	f.ttl[key] = duration
	return nil
}

func (f *fakeDatastore) Delete(key string) error {
	delete(f.kv, key)
	delete(f.ttl, key)
	return nil
}

func (f *fakeDatastore) AddToList(key string, value string) error { return nil }
func (f *fakeDatastore) GetListRange(key string, start, stop int64) ([]string, error) {
	return nil, nil
}
func (f *fakeDatastore) RemoveFromList(key string, value string) error { return nil }
func (f *fakeDatastore) GetUserPresence(uid string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (f *fakeDatastore) SetUserPresence(uid string, data map[string]interface{}) error {
	return nil
}

var _ store.Datastore = (*fakeDatastore)(nil)

func TestMentionThrottleKey(t *testing.T) {
	got := mentionThrottleKey("U123", "C456", "1234567890.000100")
	want := "mention-throttle:U123:C456:1234567890.000100"
	if got != want {
		t.Errorf("mentionThrottleKey = %q, want %q", got, want)
	}
}

func TestMentionThrottleInterval(t *testing.T) {
	t.Run("default when env unset", func(t *testing.T) {
		t.Setenv("AFK_MENTION_THROTTLE_INTERVAL", "")
		if got := mentionThrottleInterval(); got != defaultMentionThrottleInterval {
			t.Errorf("expected default %v, got %v", defaultMentionThrottleInterval, got)
		}
	})
	t.Run("uses env when valid", func(t *testing.T) {
		t.Setenv("AFK_MENTION_THROTTLE_INTERVAL", "5m")
		if got := mentionThrottleInterval(); got != 5*time.Minute {
			t.Errorf("expected 5m, got %v", got)
		}
	})
	t.Run("falls back when unparseable", func(t *testing.T) {
		t.Setenv("AFK_MENTION_THROTTLE_INTERVAL", "abc")
		if got := mentionThrottleInterval(); got != defaultMentionThrottleInterval {
			t.Errorf("expected fallback %v, got %v", defaultMentionThrottleInterval, got)
		}
	})
	t.Run("falls back when non-positive", func(t *testing.T) {
		t.Setenv("AFK_MENTION_THROTTLE_INTERVAL", "0s")
		if got := mentionThrottleInterval(); got != defaultMentionThrottleInterval {
			t.Errorf("expected fallback %v, got %v", defaultMentionThrottleInterval, got)
		}
	})
}

func TestShouldThrottleAutoResponse(t *testing.T) {
	t.Run("returns false when thread is empty", func(t *testing.T) {
		ds := newFakeDatastore()
		if shouldThrottleAutoResponse(ds, "U1", "C1", "") {
			t.Error("expected false for empty thread")
		}
	})
	t.Run("returns false when no throttle key set", func(t *testing.T) {
		ds := newFakeDatastore()
		if shouldThrottleAutoResponse(ds, "U1", "C1", "T1") {
			t.Error("expected false when key not present")
		}
	})
	t.Run("returns true when throttle key set", func(t *testing.T) {
		ds := newFakeDatastore()
		if err := ds.SetEX(mentionThrottleKey("U1", "C1", "T1"), "1", time.Minute); err != nil {
			t.Fatalf("failed to seed throttle key: %v", err)
		}
		if !shouldThrottleAutoResponse(ds, "U1", "C1", "T1") {
			t.Error("expected true when key present")
		}
	})
}

func TestMarkAutoResponseThrottled(t *testing.T) {
	t.Run("no-op when thread is empty", func(t *testing.T) {
		ds := newFakeDatastore()
		if err := markAutoResponseThrottled(ds, "U1", "C1", ""); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(ds.kv) != 0 {
			t.Errorf("expected no keys written, got %d", len(ds.kv))
		}
	})
	t.Run("writes throttle key with configured interval", func(t *testing.T) {
		t.Setenv("AFK_MENTION_THROTTLE_INTERVAL", "10m")
		ds := newFakeDatastore()
		if err := markAutoResponseThrottled(ds, "U1", "C1", "T1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		key := mentionThrottleKey("U1", "C1", "T1")
		if v, ok := ds.kv[key]; !ok || v != "1" {
			t.Errorf("expected throttle key %q to be set with value 1, got %q (ok=%v)", key, v, ok)
		}
		if ds.ttl[key] != 10*time.Minute {
			t.Errorf("expected TTL 10m, got %v", ds.ttl[key])
		}
	})
}
