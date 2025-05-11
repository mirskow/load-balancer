// Package ratelimiter реализует алгоритм ограничения скорости запросов (rate limiting)
// на основе алгоритма Token Bucket с хранением состояния в заданном репозитории.
//
// Дополнительно использует Lua-скрипты для атомарных операций над счетчиками токенов.
// Поддерживает автоматическое пополнение бакетов (refill) и предусматривает возможность индивидуальной настройки для каждого IP (клиента).
package ratelimiter

import (
	"context"
	_ "embed"
	"log"
	"strings"
	"time"

	"github.com/mirskow/load-balancer/internal/config"
	"github.com/mirskow/load-balancer/internal/repository"
)

//go:embed allow_script.lua
var allowScript string // Lua-скрипт для проверки и выдачи токена

//go:embed refill_script.lua
var refillScript string // Lua-скрипт для пополнения токенов в бакете

const (
	bucketKey = "bucket:" // Префикс для ключей состояния бакета в Redis
	configKey = "config:" // Префикс для ключей конфигурации бакета в Redis
)

type TokenBucket struct {
	client          repository.RateLimiterRepository
	defaultCapacity int
	defaultRate     int
	ttl             int64
}

func NewTokenBucket(ctx context.Context, client repository.RateLimiterRepository, cfg config.LimiterConfig) *TokenBucket {
	tb := &TokenBucket{
		client:          client,
		defaultCapacity: cfg.Capacity,
		defaultRate:     cfg.RatePerSec,
		ttl:             cfg.TTL,
	}

	go tb.refillLoop(ctx, cfg.RefillTime)

	return tb
}

// Allow проверяет, разрешен ли запрос для указанного IP-адреса, и выдает токен,
// если лимит не превышен. Возвращает true, если запрос разрешен.
func (t *TokenBucket) Allow(ctx context.Context, ip string) bool {
	key := bucketKey + ip
	configKey := configKey + ip
	now := time.Now().Unix()

	result, err := t.client.Eval(ctx, allowScript, []string{key, configKey}, now, t.ttl, t.defaultCapacity, t.defaultRate)

	if err != nil {
		log.Println("[RATE-LIMITER] allow errors:", err)
		return false
	}

	allowed, ok := result.(int64)
	return ok && allowed == 1
}

// refillLoop запускает бесконечный цикл периодического пополнения токенов во всех бакетах.
func (tb *TokenBucket) refillLoop(ctx context.Context, refillTime time.Duration) {
	t := time.NewTicker(time.Second * refillTime)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[RATE-LIMITER] Refill loop stopped")
			return
		case <-t.C:
			log.Println("[RATE-LIMITER] Starting refill bucket...")
			tb.refill(ctx)
			log.Println("[RATE-LIMITER] Refill bucket completed")
		}
	}
}

// refill выполняет пополнение токенов во всех найденных бакетах.
func (tb *TokenBucket) refill(ctx context.Context) {
	stateKeys, err := tb.client.Keys(ctx)
	if err != nil {
		log.Println("[RATE-LIMITER]", err)
		return
	}

	if len(stateKeys) == 0 {
		return
	}

	now := time.Now().Unix()
	for _, stateKey := range stateKeys {
		configKey := configKey + strings.Split(stateKey, ":")[1]

		_, err = tb.client.Eval(ctx, refillScript, []string{stateKey, configKey}, now, tb.ttl)
		if err != nil {
			log.Printf("[RATE-LIMITER] Refill %s errors: %v\n", stateKey, err)
		}
	}
}
