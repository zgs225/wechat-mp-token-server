package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
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
		mw.logger.Log("method", "GetToken", "appid", appid, "appsecret", appsecret, "token", tk, "error", err, "took", time.Since(b))
	}(time.Now())

	return mw.next.GetToken(ctx, appid, appsecret)
}

func InstrumentingMiddleware(counts metrics.Counter) Middleware {
	return func(next Service) Service {
		return &instrumentingMiddleware{
			counts: counts,
			next:   next,
		}
	}
}

type instrumentingMiddleware struct {
	counts metrics.Counter
	next   Service
}

func (mw instrumentingMiddleware) GetToken(ctx context.Context, appid, appsecret string) (tk string, err error) {
	v, err := mw.next.GetToken(ctx, appid, appsecret)
	mw.counts.With("appid", appid).Add(float64(1))
	return v, err
}
