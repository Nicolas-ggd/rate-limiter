package rrl

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// define constant variable of keyPrefix to avoid duplicate key in Redis
const keyPrefix = "ls_prefix:"

// RateLimiter is struct based on Redis
type RateLimiter struct {
	// client is redis Client
	client *redis.Client

	// number of requests a user can make to an API within a specified timeframe
	request int

	// time interval for new request
	interval time.Duration

	// logger for logging rate limit events
	logger *log.Logger
}

// encodeKey function encodes received value parameter with base64
func encodeKey(value string) string {
	return b64.StdEncoding.EncodeToString([]byte(value))
}

// NewRateLimiter to received and define new RateLimiter struct
func NewRateLimiter(client *redis.Client, request int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		client:   client,
		request:  request,
		interval: interval,
		logger:   log.New(os.Stdout, "RateLimiter: ", log.Lmicroseconds),
	}
}

// Allow function is a method of the RateLimiter struct. It is responsible for determining whether a specific request should be allowed based on the rate limiting rules.
// This function interacts with Redis to track and enforce the rate limit for a given key
//
// Parameters:
//
// key (string): A unique identifier for the request, typically representing the client making the request, such as an IP address.
//
// Returns:
//
// bool: Returns true if the request is allowed, false otherwise.
func (rl *RateLimiter) Allow(key string) bool {
	redisKey := keyPrefix + key

	// encode key
	sEnc := encodeKey(redisKey)

	val, err := rl.client.Get(context.Background(), sEnc).Int()
	if errors.Is(err, redis.Nil) {
		err = rl.client.Set(context.Background(), sEnc, rl.request-1, rl.interval).Err()
		if err != nil {
			rl.logger.Printf("Error setting key in Redis: %v", err)
			return false
		}

		return true
	} else if err != nil {
		rl.logger.Printf("Error getting key from Redis: %+v\n", err)
		return false
	}

	if val > 0 {
		_, err = rl.client.Decr(context.Background(), sEnc).Result()
		if err != nil {
			rl.logger.Printf("Error getting key from Redis: %+v\n", err)
			return false
		}

		return true
	}

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

		if !limiter.Allow(ip) {
			limiter.logger.Printf("Rate limit exceeded for IP: %s", ip)
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}

		c.Next()
	}
}
