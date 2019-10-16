package endpoint

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/zgs225/wechat-mp-token-server/pkg/service"
)

type Set struct {
	GetTokenEndpoint endpoint.Endpoint
}

func MakeGetTokenEndpoint(s service.Servce) endpoint.Endpoint {
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
