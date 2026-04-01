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

	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.NewLogger().Error("Invalid Redis URL")
		return nil, err
	}

	rdb := redis.NewClient(opts)

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		logger.NewLogger().Error("Error connecting to Redis")
		return nil, err
	}

	logger.NewLogger().Info("Connected to Redis")
	return rdb, nil
}
