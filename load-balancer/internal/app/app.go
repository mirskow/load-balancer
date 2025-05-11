// Package app предоставляет основную логику запуска и управления сервером балансировщика нагрузки.
//
// Основная цель этого пакета — инициализировать все необходимые компоненты системы:
//   - Настроить конфигурацию сервера
//   - Создать и настроить Redis-клиент
//   - Инициализировать репозитории и сервисы
//   - Запустить HTTP-сервер с обработчиками
//
// Включает также логику для плавного завершения работы сервера с обработкой сигналов остановки
// (SIGTERM, SIGINT) и корректным завершением соединений.
//
// Реализует корректное завершение работы с использованием контекста и каналов
package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mirskow/load-balancer/internal/config"
	"github.com/mirskow/load-balancer/internal/handler"
	"github.com/mirskow/load-balancer/internal/repository"
	"github.com/mirskow/load-balancer/internal/server"
	"github.com/mirskow/load-balancer/internal/services"
)

const configDir = "configs"

func Run() {
	cfg, err := config.Init(configDir)
	if err != nil {
		log.Fatalf("[MAIN] errors initialising config: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	redis, err := repository.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("[MAIN] redis client creation error: %s", err)
	}

	repos := repository.NewRepository(redis)
	services := services.NewServices(ctx, repos, *cfg)
	handlers := handler.NewHandler(services)

	srv := server.NewServer(cfg.HTTP, handlers)

	go func() {
		if err := srv.Run(); err != nil && err != http.ErrServerClosed {
			log.Println("[MAIN] Error running server:", err)
		}
	}()

	log.Println("[MAIN] Server with balancer build and run at :", cfg.HTTP.Port)

	//graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-quit
	log.Println("[MAIN] Shutdown initiated...")

	cancel()

	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("[MAIN] Error close server connection")
	}

	log.Println("[MAIN] Server gracefully stopped.")
}
