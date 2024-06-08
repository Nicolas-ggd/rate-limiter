package ls

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
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
	limiter := NewRateLimiter(client, 5, time.Minute, "testPrefix:")

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"First Request", "user1", true},
		{"Second Request", "user1", true},
		{"Third Request", "user1", true},
		{"Fourth Request", "user1", true},
		{"Fifth Request", "user1", true},
		{"Sixth Request", "user1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limiter.Allow(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
