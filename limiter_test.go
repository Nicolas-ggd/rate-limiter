package rrl

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func setupRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	client.FlushDB(context.Background()) // Clean the DB before each test
	return client
}

func TestRateLimiter_Allow(t *testing.T) {
	client := setupRedisClient()
	limiter, err := NewRateLimiter(&RateLimiter{
		Rate:           1,
		MaxTokens:      5,
		RefillInterval: 1 * time.Second,
		Client:         client,
		HashKey:        false,
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		key      string
		expected bool
		delay    time.Duration
	}{
		{"First Request", "user1", true, 0},
		{"Second Request", "user1", true, 0},
		{"Third Request", "user1", true, 0},
		{"Fourth Request", "user1", true, 0},
		{"Fifth Request", "user1", true, 0},
		{"Sixth Request", "user1", false, 0},
		{"Wait for Refill", "user1", true, 1 * time.Second},
	}

	for _, tt := range tests {
		if tt.delay > 0 {
			time.Sleep(tt.delay)
		}
		t.Run(tt.name, func(t *testing.T) {
			result := limiter.IsRequestAllowed(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
