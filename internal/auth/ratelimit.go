package auth

import (
	"sync"
	"time"
)

// rateLimiter es un contador con caducidad para limitar intentos en memoria.
// La ventana empieza con el primer intento y no se desliza con cada uno.
type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	count   int
	expires time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]rateLimitEntry),
	}
}

// allow cuenta un intento y devuelve si está permitido.
func (l *rateLimiter) allow(key string, now time.Time) bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)
	entry, ok := l.entries[key]
	if !ok || !now.Before(entry.expires) {
		l.entries[key] = rateLimitEntry{count: 1, expires: now.Add(l.window)}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

// blocked indica si la clave ya agotó sus intentos en la ventana actual.
func (l *rateLimiter) blocked(key string, now time.Time) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	l.pruneLocked(now)
	entry, ok := l.entries[key]
	return ok && now.Before(entry.expires) && entry.count >= l.limit
}

func (l *rateLimiter) reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *rateLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.entries {
		if !now.Before(entry.expires) {
			delete(l.entries, key)
		}
	}
}
