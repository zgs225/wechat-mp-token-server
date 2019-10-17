package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"

	kittransport "github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	stdopentracing "github.com/opentracing/opentracing-go"
	"github.com/zgs225/gokitkit"
	"github.com/zgs225/wechat-mp-token-server/pkg/endpoint"
)

func NewHTTPHandler(endpoints *endpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) http.Handler {
	opts := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorHandler(kittransport.NewLogErrorHandler(logger)),
	}

	if tracer != nil {
		opts = append(opts, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetToken", logger)))
	}

	m := http.NewServeMux()
	m.Handle("/get-token", httptransport.NewServer(
		endpoints.GetTokenEndpoint,
		decodeHTTPGetTokenRequest,
		gokitkit.EncodeHTTPGenericResponse,
		opts...,
	))

	return m
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	return http.StatusBadRequest
}

type errorWrapper struct {
	Error string `json:"error"`
}

func decodeHTTPGetTokenRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.GetTokenRequst
	err := json.NewDecoder(r.Body).Decode(&req)
	return &req, err
}
