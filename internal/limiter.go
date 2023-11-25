package internal

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu                   sync.Mutex
	limitsByIp           map[string]*TokenBucket
	defaultAllowRequests int
	defaultInterval      time.Duration
}

func NewRateLimiter(defaultAllowRequests int, defaultInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		limitsByIp:           make(map[string]*TokenBucket),
		defaultAllowRequests: defaultAllowRequests,
		defaultInterval:      defaultInterval,
	}
}

func (r *RateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	tokenBucket, ok := r.limitsByIp[ip]
	r.mu.Unlock()
	// Если для переданного ip еще нет ограничителя, создадим новый с параметрами по умолчанию
	if !ok {
		tokenBucket = NewTokenBucket(r.defaultAllowRequests, r.defaultInterval)
		r.mu.Lock()
		r.limitsByIp[ip] = tokenBucket
		r.mu.Unlock()

		r.runBucketMonitoring(ip, tokenBucket)
	}

	return tokenBucket.AllowTake()
}

func (r *RateLimiter) AddIp(ip string, allowRequests int, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.limitsByIp[ip]
	// Если для ip уже добавлен ограничитель, то просто ничего не делаем
	if ok {
		return
	}

	tokenBucket := NewTokenBucket(allowRequests, interval)
	r.limitsByIp[ip] = tokenBucket
	r.runBucketMonitoring(ip, tokenBucket)
}

// Для каждого нового IP выполняется мониторинг нужно ли наполнить корзину маркерами.
// Также если для данного ip не было активности более минуты, то мониторинг завершается
func (r *RateLimiter) runBucketMonitoring(ip string, tb *TokenBucket) {
	ticker := time.NewTicker(tb.InsertInterval())
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				if tb.IsFull() {
					// Если после последнего обновления корзины прошло больше минуты, то удаляем корзину для
					// данного ip и выходим из функции "наполнителя", так как данный ip адрес не активен и нет
					// смысла держать в памяти ограничитель для него.
					if time.Now().After(tb.LastTakeAt().Add(time.Minute)) {
						r.mu.Lock()
						delete(r.limitsByIp, ip)
						r.mu.Unlock()

						return
					}
					continue
				}
				// Если корзина не полная, то увеличиваем количество маркеров в корзине
				tb.Add(tb.cap)
			}
		}
	}()
}
