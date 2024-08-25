# Rate Limiter
This project showcases an implementation of a rate limiter middleware in Golang using Redis. The rate limiter restricts the number of requests a user can make to an API within a defined timeframe, helping to manage and control API usage effectively.

## Features
- Configurable Limits - Set the maximum number of requests allowed per specified timeframe dynamically.
- Redis Integration - Utilizes Redis for efficient storage and retrieval of rate limiting data, ensuring scalability and performance.
- Customizable Response - Define the response when the rate limit is exceeded.

## Installation

```shell
go get github.com/Nicolas-ggd/rate-limiter
```

## Usage
Redis-based Rate Limiter

1. Implement the Redis-based rate limiter middleware as shown:

```go
package main

import (
	"net/http"
	"time"

	"github.com/Nicolas-ggd/rate-limiter"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	r := gin.Default()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Call NewRateLimiter function from rrl package.
	// First parameter is the Redis client.
	// Second parameter is the rate (tokens per second).
	// Third parameter is the maximum number of tokens.
	// Fourth parameter is time duration, token refill is depending on x time interval
	limiter := rrl.NewRateLimiter(client, 1, 10, 30*time.Second)

	// Use RateLimiterMiddleware from rrl package and pass limiter.
	// This middleware works for all routes in your application,
	// including static files served when you open a web browser.
	r.Use(rrl.RateLimiterMiddleware(limiter))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome!"})
	})

	// Using this way allows the RateLimiterMiddleware to work for only specific routes.
	r.GET("/some", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Some!"})
	})

	r.Run(":8080")
}

```

## License
This project is licensed under the MIT License.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes ;).