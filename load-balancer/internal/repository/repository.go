// Package repository предоставляет абстракции и реализации для работы с хранилищем данных,
// используемым в сервисах, в частности - для реализации репозитория лимитирования запросов (rate limiter).
//
// Основные возможности пакета:
//   - Определяет интерфейс RateLimiterRepository для взаимодействия с хранилищем лимитов (в данной реализации используется Redis).
//   - Реализует структуру Repository, инкапсулирующую доступ к RateLimiterRepository.
//   - Предоставляет функцию NewRepository для инициализации репозитория с использованием клиента Redis.
package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RateLimiterRepository interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	Keys(ctx context.Context) ([]string, error)
}

type Repository struct {
	RateLimiterRepository RateLimiterRepository
}

func NewRepository(db *redis.Client) *Repository {
	return &Repository{
		RateLimiterRepository: NewLimiterRepository(db),
	}
}
