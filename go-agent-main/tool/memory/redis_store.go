package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/redis/go-redis/v9"
)

type redisSessionClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
}

type goRedisSessionClient struct {
	client *redis.Client
}

func (c *goRedisSessionClient) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *goRedisSessionClient) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

type persistedHistory struct {
	History   []*schema.Message `json:"history"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type RedisStore struct {
	client     redisSessionClient
	keyPrefix  string
	historyTTL time.Duration
	summaryTTL time.Duration
}

func NewRedisStore(client *redis.Client, keyPrefix string, historyTTL, summaryTTL time.Duration) *RedisStore {
	return newRedisStore(&goRedisSessionClient{client: client}, keyPrefix, historyTTL, summaryTTL)
}

func newRedisStore(client redisSessionClient, keyPrefix string, historyTTL, summaryTTL time.Duration) *RedisStore {
	keyPrefix = strings.Trim(strings.TrimSpace(keyPrefix), ":")
	if keyPrefix == "" {
		keyPrefix = "ai-blog"
	}
	if historyTTL <= 0 {
		historyTTL = 24 * time.Hour
	}
	if summaryTTL <= 0 {
		summaryTTL = 7 * 24 * time.Hour
	}
	return &RedisStore{
		client:     client,
		keyPrefix:  keyPrefix,
		historyTTL: historyTTL,
		summaryTTL: summaryTTL,
	}
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	session := &Session{ID: sessionID}

	historyJSON, err := s.client.Get(ctx, s.historyKey(sessionID))
	if err == nil {
		var record persistedHistory
		if unmarshalErr := json.Unmarshal([]byte(historyJSON), &record); unmarshalErr != nil {
			return nil, fmt.Errorf("decode session history: %w", unmarshalErr)
		}
		session.History = record.History
		session.UpdatedAt = record.UpdatedAt
	} else if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("load session history: %w", err)
	}

	summary, err := s.client.Get(ctx, s.summaryKey(sessionID))
	if err == nil {
		session.Summary = summary
	} else if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("load session summary: %w", err)
	}

	return session, nil
}

func (s *RedisStore) Save(ctx context.Context, sessionID string, session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	session.ID = sessionID
	session.UpdatedAt = time.Now()

	historyJSON, err := json.Marshal(persistedHistory{
		History:   session.History,
		UpdatedAt: session.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode session history: %w", err)
	}
	if err := s.client.Set(ctx, s.historyKey(sessionID), historyJSON, s.historyTTL); err != nil {
		return fmt.Errorf("save session history: %w", err)
	}
	if err := s.client.Set(ctx, s.summaryKey(sessionID), session.Summary, s.summaryTTL); err != nil {
		return fmt.Errorf("save session summary: %w", err)
	}
	return nil
}

func (s *RedisStore) historyKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s:history", s.keyPrefix, sessionID)
}

func (s *RedisStore) summaryKey(sessionID string) string {
	return fmt.Sprintf("%s:session:%s:summary", s.keyPrefix, sessionID)
}
