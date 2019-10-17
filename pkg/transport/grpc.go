package transport

import (
	"context"
	"errors"
	"time"

	"golang.org/x/time/rate"

	"github.com/go-kit/kit/circuitbreaker"
	kitendpoint "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/tracing/opentracing"
	kittransport "github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/sony/gobreaker"
	"github.com/zgs225/wechat-mp-token-server/pb"
	"github.com/zgs225/wechat-mp-token-server/pkg/endpoint"
	"github.com/zgs225/wechat-mp-token-server/pkg/service"
	"google.golang.org/grpc"
)

func NewGRPCServer(endpoints *endpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) pb.WechatTokenServer {
	opts := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(kittransport.NewLogErrorHandler(logger)),
	}

	return &grpcServer{
		getToken: grpctransport.NewServer(
			endpoints.GetTokenEndpoint,
			decodeGRPCGetTokenRequst,
			encodeGRPCGetTokenResponse,
			append(opts, grpctransport.ServerBefore(opentracing.GRPCToContext(tracer, "GetToken", logger)))...,
		),
	}
}

type grpcServer struct {
	getToken grpctransport.Handler
}

func (s *grpcServer) GetToken(ctx context.Context, req *pb.GetTokenRequest) (*pb.GetTokenReply, error) {
	_, rep, err := s.getToken.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.GetTokenReply), nil
}

func decodeGRPCGetTokenRequst(_ context.Context, req interface{}) (interface{}, error) {
	r := req.(*pb.GetTokenRequest)
	return &endpoint.GetTokenRequst{AppID: r.GetAppid(), AppSecret: r.GetAppsecret()}, nil
}

func encodeGRPCGetTokenResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(*endpoint.GetTokenResponse)
	return &pb.GetTokenReply{Code: 0, Token: resp.Token, Err: err2str(resp.Err)}, nil
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// NewGRPCClient returns Wechat get token service backed by a gRPC server
func NewGRPCClient(conn *grpc.ClientConn, tracer stdopentracing.Tracer, logger log.Logger) service.Service {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	var getTokenEndpoint kitendpoint.Endpoint
	{
		getTokenEndpoint = grpctransport.NewClient(
			conn,
			"pb.WechatToken",
			"GetToken",
			encodeGRPCGetTokenRequest,
			decodeGRPCGetTokenResponse,
			pb.GetTokenReply{},
			grpctransport.ClientBefore(opentracing.ContextToGRPC(tracer, logger)),
		).Endpoint()
		getTokenEndpoint = opentracing.TraceClient(tracer, "GetToken")(getTokenEndpoint)
		getTokenEndpoint = limiter(getTokenEndpoint)
		getTokenEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "GetToken",
			Timeout: 30 * time.Second,
		}))(getTokenEndpoint)
	}

	return &endpoint.Set{
		GetTokenEndpoint: getTokenEndpoint,
	}
}

func encodeGRPCGetTokenRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(*endpoint.GetTokenRequst)
	return &pb.GetTokenRequest{Appid: req.AppID, Appsecret: req.AppSecret}, nil
}

func decodeGRPCGetTokenResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.GetTokenReply)
	return &endpoint.GetTokenResponse{Token: reply.GetToken(), Err: str2error(reply.GetErr())}, nil
}

func str2error(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}
