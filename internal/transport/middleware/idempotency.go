//
// Idempotency middleware prevents duplicate processing of POST requests by caching
// successful responses (2xx) for 24 hours. clients send an Idempotency-Key header,
// and subsequent requests with the same key, path, and body hash return the cached
// response without re-executing the handler. Expired cache entries are automatically
// cleaned up hourly.
//
// the design is in-memory, but it can be migrated to Redis.

package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	IdempotencyKeyHeader = "Idempotency-Key"
	TTL                  = 24 * time.Hour
)

type cachedResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Timestamp  time.Time
}

type IdempotencyStore struct {
	cache  map[string]*cachedResponse
	mu     sync.RWMutex
	logger *slog.Logger
}

func NewIdempotencyStore(logger *slog.Logger) *IdempotencyStore {
	store := &IdempotencyStore{
		cache:  make(map[string]*cachedResponse),
		logger: logger,
	}

	// cleanup goroutine
	go store.cleanup()

	return store
}

func (s *IdempotencyStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, resp := range s.cache {
			if now.Sub(resp.Timestamp) > TTL {
				delete(s.cache, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *IdempotencyStore) Get(key string) (*cachedResponse, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp, exists := s.cache[key]
	if !exists {
		return nil, false
	}

	if time.Since(resp.Timestamp) > TTL {
		return nil, false
	}

	return resp, true
}

func (s *IdempotencyStore) Set(key string, resp *cachedResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = resp
}

// idempotency middleware - only POST requests
// takes dependencies (store)
func Idempotency(store *IdempotencyStore) func(next http.Handler) http.Handler {
	// takes the next handler in chain
	return func(next http.Handler) http.Handler {
		// the actual HTTP handler function
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get(IdempotencyKeyHeader)

			if idempotencyKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))

			bodyHash := sha256.Sum256(body)
			cacheKey := fmt.Sprintf("%s:%s:%s", idempotencyKey, r.URL.Path, hex.EncodeToString(bodyHash[:]))

			if cached, exists := store.Get(cacheKey); exists {
				store.logger.Info("idempotency cache hit",
					slog.String("key", idempotencyKey),
					slog.String("path", r.URL.Path),
				)

				for k, v := range cached.Headers {
					w.Header()[k] = v
				}

				w.WriteHeader(cached.StatusCode)
				w.Write(cached.Body)
				return
			}

			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(recorder, r)

			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				store.Set(cacheKey, &cachedResponse{
					StatusCode: recorder.statusCode,
					Body:       recorder.body.Bytes(),
					Headers:    recorder.Header().Clone(),
					Timestamp:  time.Now(),
				})

				store.logger.Info("idempotency response cached",
					slog.String("key", idempotencyKey),
					slog.String("path", r.URL.Path),
				)
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
