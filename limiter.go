package rrl

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	rate int64

	// represents the max tokens capacity that the bucket can hold
	maxTokens int64

	// tokens currently present in the bucket at any time
	currentToken int64

	// lastRefillTime represents time that this bucket fill operation was tried
	refillInterval time.Duration

	mutex sync.Mutex

	// client is redis Client
	client *redis.Client

	// logger for logging rate limit events
	logger *log.Logger
}

// encodeKey function encodes received value parameter with base64
func encodeKey(value string) string {
	return b64.StdEncoding.EncodeToString([]byte(value))
}

// NewRateLimiter to received and define new RateLimiter struct
func NewRateLimiter(client *redis.Client, rate, maxToken int64, refillInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		client:         client,
		rate:           rate,
		maxTokens:      maxToken,
		refillInterval: refillInterval,
		currentToken:   maxToken,
		logger:         log.New(os.Stdout, "RateLimiter: ", log.Lmicroseconds),
	}
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
func (rl *RateLimiter) IsRequestAllowed(key string, token int64) bool {
	// use mutex to avoid race condition
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	sEnc := keyPrefix + encodeKey(key)

	tokenCount, err := rl.client.Get(context.Background(), sEnc).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		rl.logger.Printf("Error getting token count from Redis: %v", err)
		return false
	}

	if errors.Is(err, redis.Nil) {
		tokenCount = rl.maxTokens
	}

	lastRefillTimeStr, err := rl.client.Get(context.Background(), sEnc+lastRefillPrefix).Result()
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

	if tokenCount >= token {
		tokenCount -= token
		rl.client.Set(context.Background(), sEnc, tokenCount, 0)
		rl.client.Set(context.Background(), sEnc+lastRefillPrefix, time.Now().Format(time.RFC3339), 0)
		return true
	}

	rl.client.Set(context.Background(), sEnc+lastRefillPrefix, time.Now().Format(time.RFC3339), 0)
	return false
}

// RateLimiterMiddleware function is a middleware for the Gin web framework that enforces rate limiting on incoming requests.
// This middleware uses a RateLimiter instance to track and limit the number of requests a client can make within a specified time interval.
//
// Parameters:
//
// limiter (*RateLimiter): An instance of the RateLimiter struct that defines the rate limiting rules and interacts with Redis to enforce them.
//
// Returns:
//
// gin.HandlerFunc: A Gin handler function that can be used as middleware in the Gin router.
func RateLimiterMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		token := int64(1)
		if !limiter.IsRequestAllowed(ip, token) {
			limiter.logger.Printf("Rate limit exceeded for IP: %s", ip)
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) refill(currentTokens int64, lastRefillTime time.Time) int64 {
	now := time.Now()
	elapsed := now.Sub(lastRefillTime)

	// calculate time which each token needs to refill in token bucket
	tokensToAdd := elapsed.Nanoseconds() / rl.refillInterval.Nanoseconds()
	newTokens := int64(math.Min(float64(currentTokens+tokensToAdd), float64(rl.maxTokens)))

	return newTokens
}
