# GIN Rate Limiter

This project demonstrates how to implement a rate limiter middleware in Golang using the Gin. The middleware limits the number of requests a user can make to an API within a specified timeframe.

## Features
- Configurable Limits - Set the number of requests allowed per timeframe
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

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/Nicolas-ggd/rate-limiter"
)

func main() {
    r := gin.Default()

    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Call NewRateLimiter function from ls package, first parameter is Redis client
    // second parameter is quantity of request user can in certain timeframe,
    // third parameter is time type
    limiter := rrl.NewRateLimiter(client, 10, time.Minute)

    // You can call RateLimiterMiddleware middleware from ls package and pass limiter
    r.Use(rrl.RateLimiterMiddleware(limiter))

    r.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "Welcome!"})
    })

    r.Run(":8080")
}

```

## License
This project is licensed under the MIT License.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes ;).