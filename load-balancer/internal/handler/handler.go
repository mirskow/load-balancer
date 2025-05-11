// Package handler предоставляет HTTP-обработчик для входящих запросов,
// реализует ограничение скорости (rate limiting) по IP-адресу клиента
// и маршрутизацию запросов через балансировщик нагрузки.
//
// Основные функции пакета:
//   - Извлечение IP-адреса клиента из запроса.
//   - Проверка лимита запросов с помощью сервиса RateLimiter.
//   - Передача запроса на обработку балансировщику нагрузки (LoadBalancer).
//   - Формирование стандартизированных JSON-ответов при ошибках.
package handler

import (
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/mirskow/load-balancer/internal/services"
)

type Handler struct {
	services *services.Services
}

func NewHandler(services *services.Services) *Handler {
	return &Handler{
		services: services,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP, err := getClientIP(r.RemoteAddr)
	if err != nil {
		log.Printf("[HANDLER] Error parsing remoteAddr: %s", err)
		writeJSON(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !h.services.RateLimiter.Allow(r.Context(), clientIP) {
		log.Printf("[RATE - LIMITER] Too many request from client: %s\n", clientIP)
		writeJSON(w, http.StatusTooManyRequests, "Too many request from your IP")
		return
	}

	h.services.LoadBalancer.Route(w, r)
}

func getClientIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", err
	}
	return host, nil
}

func writeJSON(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]any{
		"status":  statusCode,
		"message": message,
	}

	json.NewEncoder(w).Encode(response)
}
