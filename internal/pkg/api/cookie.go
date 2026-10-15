package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

type (
	cookieStoreKey struct{}
	requestKey     struct{}
)

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

func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestKey{}, r)
}

func GetRequest(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestKey{}).(*http.Request)
	return r, ok
}

var ErrNoRequest = errors.New("no http request in context")

func GetCookieValue(ctx context.Context, name string) (string, error) {
	r, ok := GetRequest(ctx)
	if !ok {
		return "", ErrNoRequest
	}
	c, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

func GetCookieStore(ctx context.Context) *cookieStore {
	return cookieStoreFromContext(ctx)
}
