package rrl

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"log"
	"net/http"
	"os"
	"time"
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

// NewRateLimiter to received and define new RateLimiter struct
func NewRateLimiter(client *redis.Client, request int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		client:   client,
		request:  request,
		interval: interval,
		logger:   log.New(os.Stdout, "RateLimiter: ", log.Lmicroseconds),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	redisKey := keyPrefix + key

	val, err := rl.client.Get(context.Background(), redisKey).Int()
	if errors.Is(err, redis.Nil) {
		err = rl.client.Set(context.Background(), redisKey, rl.request-1, rl.interval).Err()
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
		_, err = rl.client.Decr(context.Background(), redisKey).Result()
		if err != nil {
			rl.logger.Printf("Error getting key from Redis: %+v\n", err)
			return false
		}

		return true
	}

	return false
}

func RateLimiterMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			limiter.logger.Printf("Rate limit exceeded for IP: %s", ip)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}

		c.Next()
	}
}
