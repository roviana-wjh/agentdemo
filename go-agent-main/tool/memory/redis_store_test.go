package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

type fakeRedisValue struct {
	value      string
	expiration time.Duration
}

type fakeRedisSessionClient struct {
	values map[string]fakeRedisValue
}

func newFakeRedisSessionClient() *fakeRedisSessionClient {
	return &fakeRedisSessionClient{values: make(map[string]fakeRedisValue)}
}

func (f *fakeRedisSessionClient) Get(ctx context.Context, key string) (string, error) {
	value, ok := f.values[key]
	if !ok {
		return "", redis.Nil
	}
	return value.value, nil
}

func (f *fakeRedisSessionClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	var encoded string
	switch typed := value.(type) {
	case []byte:
		encoded = string(typed)
	case string:
		encoded = typed
	default:
		encoded = fmt.Sprint(typed)
	}
	f.values[key] = fakeRedisValue{value: encoded, expiration: expiration}
	return nil
}

func TestRedisStoreSaveAndGet(t *testing.T) {
	client := newFakeRedisSessionClient()
	store := newRedisStore(client, "test-blog", time.Hour, 2*time.Hour)
	session := &Session{
		ID:      "session-1",
		History: []*schema.Message{schema.UserMessage("hello")},
		Summary: "user likes Go",
	}

	if err := store.Save(context.Background(), session.ID, session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(loaded.History) != 1 || loaded.History[0].Content != "hello" {
		t.Fatalf("loaded history = %+v", loaded.History)
	}
	if loaded.Summary != "user likes Go" {
		t.Fatalf("loaded summary = %q", loaded.Summary)
	}
	if client.values[store.historyKey(session.ID)].expiration != time.Hour {
		t.Fatalf("history TTL was not applied")
	}
	if client.values[store.summaryKey(session.ID)].expiration != 2*time.Hour {
		t.Fatalf("summary TTL was not applied")
	}
}

func TestRedisStoreReturnsEmptySessionWhenKeysDoNotExist(t *testing.T) {
	store := newRedisStore(newFakeRedisSessionClient(), "test-blog", time.Hour, time.Hour)

	loaded, err := store.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.ID != "missing" || len(loaded.History) != 0 || loaded.Summary != "" {
		t.Fatalf("loaded session = %+v", loaded)
	}
}
