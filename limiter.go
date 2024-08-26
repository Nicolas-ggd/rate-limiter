package rrl

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// define constant variable of keyPrefix to avoid duplicate key in Redis
const (
	keyPrefix        = "ls_prefix:"
	lastRefillPrefix = "_lastRefillTime"
)

// RateLimiter is struct based on Redis
type RateLimiter struct {
	// represents the rate at which the bucket should be filled
	Rate int64

	// represents the max tokens capacity that the bucket can hold
	MaxTokens int64

	// tokens currently present in the bucket at any time
	currentToken int64

	// lastRefillTime represents time that this bucket fill operation was tried
	RefillInterval time.Duration

	// client is redis Client
	Client *redis.Client

	// decide to hash redis key
	HashKey bool

	// logger for logging rate limit events
	logger *log.Logger

	mutex sync.Mutex
}

// encodeKey function encodes received value parameter with base64
func encodeKey(value string) string {
	return b64.StdEncoding.EncodeToString([]byte(value))
}

// NewRateLimiter to received and define new RateLimiter struct
func NewRateLimiter(config *RateLimiter) (*RateLimiter, error) {
	return config, nil
}

// IsRequestAllowed function is a method of the RateLimiter struct. It is responsible for determining whether a specific request should be allowed based on the rate limiting rules.
// This function interacts with Redis to track and enforce the rate limit for a given key
//
// Parameters:
//
// key (string): A unique identifier for the request, typically representing the client making the request, such as an IP address.
//
// Returns:
//
// bool: Returns true if the request is allowed, false otherwise.
func (rl *RateLimiter) IsRequestAllowed(key string) bool {
	// use mutex to avoid race condition
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	var sEnc string
	if rl.HashKey {
		sEnc = keyPrefix + encodeKey(key)
	} else {
		sEnc = keyPrefix + key
	}

	tokenCount, err := rl.Client.Get(context.Background(), sEnc).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		rl.logger.Printf("Error getting token count from Redis: %v", err)
		return false
	}

	if errors.Is(err, redis.Nil) {
		tokenCount = rl.MaxTokens
	}

	lastRefillTimeStr, err := rl.Client.Get(context.Background(), sEnc+lastRefillPrefix).Result()
	var lastRefillTime time.Time
	if err == nil {
		lastRefillTime, err = time.Parse(time.RFC3339, lastRefillTimeStr)
		if err != nil {
			rl.logger.Printf("Error parsing last refill time from Redis: %v", err)
			return false
		}
	} else if !errors.Is(err, redis.Nil) {
		rl.logger.Printf("Error getting last refill time from Redis: %v", err)
		return false
	} else {
		lastRefillTime = time.Now()
	}

	tokenCount = rl.refill(tokenCount, lastRefillTime)

	if tokenCount >= rl.Rate {
		tokenCount -= rl.Rate
		rl.Client.Set(context.Background(), sEnc, tokenCount, 0)
		rl.Client.Set(context.Background(), sEnc+lastRefillPrefix, time.Now().Format(time.RFC3339), 0)
		return true
	}

	rl.Client.Set(context.Background(), sEnc+lastRefillPrefix, time.Now().Format(time.RFC3339), 0)
	return false
}

func (rl *RateLimiter) refill(currentTokens int64, lastRefillTime time.Time) int64 {
	now := time.Now()
	elapsed := now.Sub(lastRefillTime)

	// calculate time which each token needs to refill in token bucket
	tokensToAdd := elapsed.Nanoseconds() / rl.RefillInterval.Nanoseconds()
	newTokens := int64(math.Min(float64(currentTokens+tokensToAdd), float64(rl.MaxTokens)))

	return newTokens
}
