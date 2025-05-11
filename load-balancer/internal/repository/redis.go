package repository

import (
	"context"
	"fmt"

	"github.com/mirskow/load-balancer/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("[REDIS] can not connect with Redis: %s", err)
	}

	return client, nil
}
