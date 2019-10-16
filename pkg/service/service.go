package service

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/silenceper/wechat"
	"github.com/silenceper/wechat/cache"
)

type Service interface {
	GetToken(ctx context.Context, appid, appsecret string) (string, error)
}

func New(c cache.Cache, log log.Logger) Service {
	s := NewBasicService(c)
	s = LoggingMiddleware(log)(s)
	return s
}

type basicService struct {
	cache cache.Cache
}

func NewBasicService(c cache.Cache) Service {
	return &basicService{
		cache: c,
	}
}

func (s *basicService) GetToken(_ context.Context, appid, appsecret string) (string, error) {
	wc := wechat.NewWechat(&wechat.Config{
		AppID:     appid,
		AppSecret: appsecret,
		Cache:     s.cache,
	})

	return wc.Context.GetAccessToken()
}
