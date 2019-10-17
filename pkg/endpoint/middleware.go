package endpoint

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
)

func InstrumentingMiddleware(duration metrics.Histogram) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(b time.Time) {
				duration.With("success", fmt.Sprint(err == nil)).Observe(time.Since(b).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}
