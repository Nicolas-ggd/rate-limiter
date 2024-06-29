package rrl

import (
	"context"
	"github.com/go-redis/redis/v8"
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
	limiter := NewRateLimiter(client, 1, 5, time.Second)

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
		{"Wait for Refill", "user1", true, 5 * time.Second},
	}

	for _, tt := range tests {
		if tt.delay > 0 {
			time.Sleep(tt.delay)
		}
		t.Run(tt.name, func(t *testing.T) {
			result := limiter.IsRequestAllowed(tt.key, 1)
			assert.Equal(t, tt.expected, result)
		})
	}
}
