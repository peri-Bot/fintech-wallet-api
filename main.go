package main

import (
	"fmt"
	"log"

	"fintech_wallet_api/config"
	"fintech_wallet_api/database"
	"fintech_wallet_api/handlers"
	"fintech_wallet_api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// Initialize PostgreSQL (primary + replica).
	database.Init(cfg)

	// Initialize Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	// Set up Gin router.
	r := gin.Default()

	// Health check (no rate limiting).
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes — rate limited.
	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimiter(rdb))
	{
		v1.POST("/deposit", handlers.Deposit)
		v1.POST("/withdraw", handlers.Withdraw)
		v1.GET("/balance/:user_id", handlers.GetBalance)
	}

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("starting wallet API on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
