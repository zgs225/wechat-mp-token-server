package endpoint

import (
	"context"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/tracing/zipkin"
	stdzipkin "github.com/openzipkin/zipkin-go"
	"github.com/sony/gobreaker"
	"github.com/zgs225/wechat-mp-token-server/pkg/service"
)

type Set struct {
	GetTokenEndpoint endpoint.Endpoint
}

func New(svc service.Service, duration metrics.Histogram, tracer *stdzipkin.Tracer) *Set {
	var getTokenEndpoint endpoint.Endpoint
	{
		getTokenEndpoint = MakeGetTokenEndpoint(svc)
		getTokenEndpoint = circuitbreaker.Gobreaker(gobreaker.Settings{})(getTokenEndpoint)
		if tracer != nil {
			getTokenEndpoint = zipkin.TraceEndpoint(tracer, "GetToken")(getTokenEndpoint)
		}
		getTokenEndpoint = InstrumentingMiddleware(duration.With("method", "GetToken"))(getTokenEndpoint)
	}

	return &Set{
		GetTokenEndpoint: getTokenEndpoint,
	}
}

func (s Set) GetToken(ctx context.Context, appid, appsecret string) (string, error) {
	resp, err := s.GetTokenEndpoint(ctx, &GetTokenRequst{AppID: appid, AppSecret: appsecret})
	if err != nil {
		return "", err
	}
	r := resp.(*GetTokenResponse)
	return r.Token, r.Err
}

func MakeGetTokenEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*GetTokenRequst)
		token, err := s.GetToken(ctx, req.AppID, req.AppSecret)
		return &GetTokenResponse{Token: token, Err: err}, nil
	}
}

var (
	_ endpoint.Failer = GetTokenResponse{}
)

type GetTokenRequst struct {
	AppID, AppSecret string
}

type GetTokenResponse struct {
	Token string `json:"token"`
	Err   error  `json:"-"`
}

func (r GetTokenResponse) Failed() error { return r.Err }
