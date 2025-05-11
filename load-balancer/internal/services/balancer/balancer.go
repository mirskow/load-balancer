// Package balancer реализует балансировщик нагрузки для HTTP-запросов с поддержкой различных стратегий
// распределения, проверки состояния (health check) и автоматического исключения неработающих бэкендов.
//
// Основные возможности пакета:
//   - Инкапсулирует пул бэкендов (serverPool) и стратегию выбора следующего бэкенда (BalancingStrategy).
//   - Поддерживает автоматическую проверку состояния бэкендов (health check) с заданным интервалом.
//   - Использует ReverseProxy для прозрачной передачи запросов на выбранный бэкенд.
//   - Автоматически помечает бэкенд как "нерабочий" при ошибках проксирования.
//   - Позволяет гибко расширять стратегии балансировки (например, Round Robin, Least Connections и др.).
package balancer

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/mirskow/load-balancer/internal/backends"
	"github.com/mirskow/load-balancer/internal/config"
)

type ctxKey string

const backendKey ctxKey = "backend"

// BalancingStrategy определяет интерфейс для стратегий выбора следующего бэкенда.
type BalancingStrategy interface {
	NextBackend([]*backends.Backend) *backends.Backend
}

type LoadBalancer struct {
	serverPool []*backends.Backend
	strategy   BalancingStrategy
}

// NewLoadBalancer создает новый балансировщик нагрузки с заданной конфигурацией и стратегией.
// Запускает цикл health check для проверки состояния бэкендов
func NewLoadBalancer(ctx context.Context, cfg config.BalancerConfig) *LoadBalancer {
	const startIndex uint64 = 1

	backendList := make([]*backends.Backend, 0, len(cfg.Backends))

	for _, addr := range cfg.Backends {
		serverURL, err := url.Parse(addr)
		if err != nil {
			log.Printf("[BALANCER] Error parsing backend url - %s: %v", addr, err)
			continue
		}

		proxy := createReverseProxy(serverURL)

		backendList = append(backendList, backends.NewBackend(serverURL, proxy))
	}

	lb := &LoadBalancer{
		serverPool: backendList,
		strategy:   NewRoundRobin(uint64(startIndex)),
	}

	go lb.healthCheckLoop(ctx, cfg.HealthCheckTime)

	return lb
}

// Route выбирает живой бэкенд и согласно стратегии проксирует запрос к нему.
// Если нет доступных бэкендов, возвращает ошибку 503.
func (lb *LoadBalancer) Route(w http.ResponseWriter, r *http.Request) {
	aliveBackends := lb.getAliveBackends()
	if len(aliveBackends) == 0 {
		lb.respondNoBackends(w)
		return
	}

	backend := lb.strategy.NextBackend(aliveBackends)
	if backend == nil {
		lb.respondNoBackends(w)
		return
	}

	ctx := context.WithValue(r.Context(), backendKey, backend)
	r = r.WithContext(ctx)

	log.Printf("[BALANCER] Forwarding request to: %s\n", backend.URL.String())
	backend.ReverseProxy.ServeHTTP(w, r)
}

func (lb *LoadBalancer) respondNoBackends(w http.ResponseWriter) {
	http.Error(w, "Service unavailable: no alive backend", http.StatusServiceUnavailable)
	log.Println("[BALANCER] No alive backends available")
}

func (lb *LoadBalancer) getAliveBackends() []*backends.Backend {
	alive := make([]*backends.Backend, 0, len(lb.serverPool))

	for _, b := range lb.serverPool {
		if b.IsAlive() {
			alive = append(alive, b)
		}
	}

	return alive
}

func (lb *LoadBalancer) healthCheck() {
	for _, b := range lb.serverPool {
		if b == nil || b.URL == nil {
			log.Printf("[BALANCER - HealthCheckError] nil backend or URL")
			continue
		}

		resp, err := http.Get(b.URL.String())
		alive := err == nil && resp != nil && resp.StatusCode == http.StatusOK

		b.SetAlive(alive)

		if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
			code := 0
			if resp != nil {
				code = resp.StatusCode
			}
			log.Printf("[BALANCER - HealthCheckError] %s - %d : (%v)\n", b.URL, code, err)
		}
	}

}

// healthCheckLoop запускает периодическую проверку состояния бэкендов до завершения контекста
func (lb *LoadBalancer) healthCheckLoop(ctx context.Context, healthCheckTime time.Duration) {
	t := time.NewTicker(time.Second * healthCheckTime)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[BALANCER] Health check loop stopped")
			return
		case <-t.C:
			log.Println("[BALANCER] Starting health check...")
			lb.healthCheck()
			log.Println("[BALANCER] Health check completed")
		}
	}
}

// newReverseProxy создает reverse proxy для заданного backend URL с кастомным обработчиком ошибок.
func createReverseProxy(serverURL *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(serverURL)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		backend, ok := r.Context().Value(backendKey).(*backends.Backend)

		if ok {
			backend.SetAlive(false)
			log.Printf("[BALANCER - ErrorHandler] Marked backend %s as DOWN: %v", backend.URL, err)
		}

		http.Error(w, "Backend unavailable", http.StatusServiceUnavailable)
	}

	return proxy
}
