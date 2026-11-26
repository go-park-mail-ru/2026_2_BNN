package cookies

import (
	"context"
	"net/http"
)

type (
	storeKey   struct{}
	requestKey struct{}
)

type store struct {
	cookies []*http.Cookie
}

func Set(ctx context.Context, c *http.Cookie) {
	if s, ok := ctx.Value(storeKey{}).(*store); ok {
		s.cookies = append(s.cookies, c)
	}
}

func Get(ctx context.Context, name string) (string, error) {
	r, ok := ctx.Value(requestKey{}).(*http.Request)
	if !ok {
		return "", http.ErrNoCookie
	}

	c, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	return c.Value, nil
}
