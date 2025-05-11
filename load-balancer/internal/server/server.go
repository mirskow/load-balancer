// Package server реализует HTTP-сервер для запуска и управления жизненным циклом приложения.
//
// Основные возможности пакета:
//   - Инкапсулирует создание и настройку http.Server на основе конфигурации.
//   - Предоставляет методы для запуска сервера (Run) и его корректной остановки (Stop) с поддержкой graceful shutdown.
//   - Позволяет передавать кастомный http.Handler для обработки входящих HTTP-запросов.
package server

import (
	"context"
	"net/http"

	"github.com/mirskow/load-balancer/internal/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg config.HTTPConfig, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
