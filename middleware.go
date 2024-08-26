package rrl

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

const (
	HeaderRateLimit          = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
)

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
		if !limiter.IsRequestAllowed(ip) {
			log.Printf("Rate limit exceeded for IP: %s", ip)
			c.Header(HeaderRateLimitRemaining, strconv.Itoa(int(limiter.currentToken)))
			c.Header(HeaderRateLimit, strconv.Itoa(int(limiter.MaxTokens)))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			c.Abort()
			return
		}

		c.Next()
	}
}
