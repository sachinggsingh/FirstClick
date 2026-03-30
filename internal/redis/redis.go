package redis

import (
	"context"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/sachinggsingh/firstclick/internal/logger"
)

func ConnectToRedis() (*redis.Client, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		// Password: strings.TrimSpace(os.Getenv("REDIS_PASS")),
		// DB: 0,
	})

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.NewLogger().Error("Error in Connecting with REDIS")
		return nil, err
	}
	logger.NewLogger().Info("Connected to Redis")
	return rdb, nil
}
