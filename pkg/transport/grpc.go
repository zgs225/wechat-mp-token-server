package transport

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	kittransport "github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/zgs225/wechat-mp-token-server/pb"
	"github.com/zgs225/wechat-mp-token-server/pkg/endpoint"
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
