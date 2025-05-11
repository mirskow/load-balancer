// Реализация стратегии балансировки нагрузки Round Robin.
//
// Стратегия Round Robin равномерно распределяет входящие запросы между доступными бэкендами.
// Для потокобезопасности используется атомарный счетчик текущего индекса.
//
// Особенности реализации:
//   - Атомарное обновление индекса для поддержки конкурентного доступа.
//   - Пропуск неработающих (неживых) бэкендов при выборе следующего.
//   - Если все бэкенды недоступны - возвращается nil.
package balancer

import (
	"sync/atomic"

	"github.com/mirskow/load-balancer/internal/backends"
)

type RoundRobin struct {
	currentIndex uint64
}

// NewRoundRobin создает новый экземпляр RoundRobin с заданным стартовым индексом.
func NewRoundRobin(currentIndex uint64) *RoundRobin {
	return &RoundRobin{
		currentIndex: currentIndex,
	}
}

// getNextIndex возвращает следующий индекс для выбора бэкенда (с учетом количества бэкендов).
func (rr *RoundRobin) getNextIndex(countBackends int) int {
	return int(atomic.AddUint64(&rr.currentIndex, uint64(1)) % uint64(countBackends))
}

// NextBackend выбирает следующий живой бэкенд согласно стратегии Round Robin.
func (rr *RoundRobin) NextBackend(backends []*backends.Backend) *backends.Backend {
	countBackends := len(backends)
	nextIndex := rr.getNextIndex(countBackends)
	end := countBackends + nextIndex

	for i := nextIndex; i < end; i++ {
		index := i % countBackends
		if backends[index].IsAlive() {
			if i != nextIndex {
				atomic.StoreUint64(&rr.currentIndex, uint64(index))
			}
			return backends[index]
		}
	}
	return nil
}
