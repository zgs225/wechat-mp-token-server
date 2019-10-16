package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
)

type Middleware func(Service) Service

func LoggingMiddleware(l log.Logger) Middleware {
	return func(next Service) Service {
		return &loggingMiddleware{l, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) GetToken(ctx context.Context, appid, appsecret string) (tk string, err error) {
	defer func(b time.Time) {
		mw.logger.Log("method", "GetToken", "appid", appid, "appsecret", appsecret, "token", tk, "error", err, "duration", time.Since(b))
	}(time.Now())

	return mw.next.GetToken(ctx, appid, appsecret)
}
