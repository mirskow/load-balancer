// LimiterRepository предоставляет методы для работы с rate limiter в Redis.
//
// Основные возможности LimiterRepository:
//   - Выполнение Lua-скриптов в Redis (Eval).
//   - Получение всех ключей bucket:* (Keys).
package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type LimiterRepository struct {
	client *redis.Client
}

func NewLimiterRepository(client *redis.Client) *LimiterRepository {
	return &LimiterRepository{client: client}
}

func (r *LimiterRepository) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.client.Eval(ctx, script, keys, args...).Result()
}

func (r *LimiterRepository) Keys(ctx context.Context) ([]string, error) {
	return r.client.Keys(ctx, "bucket:*").Result()
}
