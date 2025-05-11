// Package services агрегирует и инициализирует основные сервисы приложения, такие как
// ограничение скорости запросов (rate limiter) и балансировщик нагрузки (load balancer).
//
// Основные возможности пакета:
//   - Определяет интерфейсы для сервисов RateLimiter и Balancer, упрощающие тестирование и масштабирование.
//   - Реализует структуру Services, объединяющую все сервисы приложения для удобной передачи по слоям.
//   - Предоставляет функцию NewServices для инициализации сервисов на основе репозиториев и конфигурации.
package services

import (
	"context"
	"net/http"

	"github.com/mirskow/load-balancer/internal/config"
	"github.com/mirskow/load-balancer/internal/repository"
	"github.com/mirskow/load-balancer/internal/services/balancer"
	ratelimiter "github.com/mirskow/load-balancer/internal/services/rate-limiter"
)

type RateLimiter interface {
	Allow(ctx context.Context, ip string) bool
}

type Balancer interface {
	Route(http.ResponseWriter, *http.Request)
}

type Services struct {
	RateLimiter  RateLimiter
	LoadBalancer Balancer
}

func NewServices(ctx context.Context, repo *repository.Repository, cfg config.Config) *Services {
	return &Services{
		RateLimiter:  ratelimiter.NewTokenBucket(ctx, repo.RateLimiterRepository, cfg.Limiter),
		LoadBalancer: balancer.NewLoadBalancer(ctx, cfg.Balancer),
	}
}
