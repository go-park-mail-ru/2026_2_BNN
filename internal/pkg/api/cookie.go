package api

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type cookieStoreKey struct{}

type Cookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type cookieStore struct {
	mu      sync.Mutex
	cookies []Cookie
}

func (s *cookieStore) add(c Cookie) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookies = append(s.cookies, c)
}

func (s *cookieStore) all() []Cookie {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Cookie, len(s.cookies))
	copy(out, s.cookies)
	return out
}

func withCookieStore(ctx context.Context) (context.Context, *cookieStore) {
	store := &cookieStore{}
	return context.WithValue(ctx, cookieStoreKey{}, store), store
}

func cookieStoreFromContext(ctx context.Context) *cookieStore {
	store, _ := ctx.Value(cookieStoreKey{}).(*cookieStore)
	return store
}

func SetCookie(ctx context.Context, c Cookie) {
	if store := cookieStoreFromContext(ctx); store != nil {
		store.add(c)
	}
}
