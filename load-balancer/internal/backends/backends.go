// Package backends реализует структуру для представления бекенд-узлов, которые используются
// в качестве целей для проксирования запросов в балансировщике нагрузки.
//
// Каждый бэкенд содержит информацию о своем URL, состоянии "живости" (Alive),
// а также обратный прокси для обработки запросов, направленных на данный бэкенд.
package backends

import (
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        atomic.Bool
	ReverseProxy *httputil.ReverseProxy
}

func NewBackend(url *url.URL, proxy *httputil.ReverseProxy) *Backend {
	b := &Backend{
		URL:          url,
		ReverseProxy: proxy,
	}

	b.Alive.Store(true)
	return b
}

// Возвращает true, если бэкенд считается живым (Alive = true), иначе false.
func (b *Backend) IsAlive() (alive bool) {
	return b.Alive.Load()
}

// Используется для обновления флага Alive в случае изменения состояния бэкенда.
func (b *Backend) SetAlive(alive bool) {
	b.Alive.Store(alive)
}
