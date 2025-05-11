// Package tests содержит интеграционные тесты для проверки производительности
// и корректности работы компонентов балансировщика нагрузки.
package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/mirskow/load-balancer/internal/config"
	"github.com/mirskow/load-balancer/internal/handler"
	"github.com/mirskow/load-balancer/internal/repository"
	"github.com/mirskow/load-balancer/internal/server"
	"github.com/mirskow/load-balancer/internal/services"
	"github.com/redis/go-redis/v9"
)

func BenchmarkAppPerformance(b *testing.B) {
	// Создаём тестовые бэкенды
	backends := createTestBackends(3)
	defer func() {
		for _, backend := range backends {
			backend.Close()
		}
	}()

	// Инициализация конфигурации
	cfg, err := config.Init("../configs/test")
	if err != nil {
		b.Fatalf("Error initializing configuration: %v", err)
	}

	// Настройка тестового окружения
	port := getFreePort(b)
	cfg.HTTP.Port = strconv.Itoa(port)
	cfg.Balancer.Backends = backendURLs(backends)

	// Инициализация Redis
	redisClient := setupTestRedis(b, cfg)
	defer redisClient.Close()

	// Создание и запуск сервера
	srv := setupTestServer(b, cfg, redisClient)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Stop(ctx); err != nil {
			b.Errorf("Server shutdown error: %v", err)
		}
	}()

	// Ожидание готовности сервера
	waitForServerReady(b, cfg.HTTP.Port)

	// Бенчмарк
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		for pb.Next() {
			resp, err := client.Get(fmt.Sprintf("http://localhost:%s", cfg.HTTP.Port))
			if err != nil {
				b.Fatalf("Request error: %v", err)
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
				b.Fatalf("Unexpected status code: %d", resp.StatusCode)
			}

			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				b.Fatalf("Error reading response body: %v", err)
			}
			resp.Body.Close()
		}
	})
}

// Вспомогательные функции для подготовки тестовой среды

func createTestBackends(n int) []*httptest.Server {
	var servers []*httptest.Server
	for i := 0; i < n; i++ {
		s := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "Hello from backend %d", i)
			}))
		servers = append(servers, s)
	}
	return servers
}

func getFreePort(b *testing.B) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		b.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func setupTestRedis(b *testing.B, cfg *config.Config) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Host + ":" + cfg.Redis.Port,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		b.Fatalf("Redis connection error: %v", err)
	}

	if err := redisClient.FlushDB(context.Background()).Err(); err != nil {
		b.Fatalf("Redis flush error: %v", err)
	}

	return redisClient
}

func setupTestServer(b *testing.B, cfg *config.Config, redisClient *redis.Client) *server.Server {
	repos := repository.NewRepository(redisClient)
	services := services.NewServices(context.Background(), repos, *cfg)
	handlers := handler.NewHandler(services)
	srv := server.NewServer(cfg.HTTP, handlers)

	go func() {
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			b.Fatalf("Server startup error: %v", err)
		}
	}()

	return srv
}

func waitForServerReady(b *testing.B, port string) {
	client := &http.Client{Timeout: 100 * time.Millisecond}
	start := time.Now()

	for {
		if time.Since(start) > 5*time.Second {
			b.Fatal("Server failed to start within 5 seconds")
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%s/", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func backendURLs(backends []*httptest.Server) []string {
	var urls []string
	for _, s := range backends {
		urls = append(urls, s.URL)
	}
	return urls
}
