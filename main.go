package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/oklog/oklog/pkg/group"
	stdopentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/silenceper/wechat/cache"
	"github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"
	"github.com/zgs225/wechat-mp-token-server/pb"
	"github.com/zgs225/wechat-mp-token-server/pkg/endpoint"
	"github.com/zgs225/wechat-mp-token-server/pkg/service"
	"github.com/zgs225/wechat-mp-token-server/pkg/transport"
	"google.golang.org/grpc"
)

func main() {
	fs := flag.NewFlagSet("wechattokensvc", flag.PanicOnError)

	var (
		debugAddr = fs.String("debug.addr", ":8080", "Debug and metrics listen address")
		httpAddr  = fs.String("http.addr", ":8081", "HTTP listen address")
		grpcAddr  = fs.String("grpc.addr", ":8082", "gRPC listen address")
		redisAddr = fs.String("redis.addr", "localhost:6379", "Redis connect addr")
		redisPass = fs.String("redis.password", "", "Redis password")
		redisDB   = fs.Int("redis.db", 0, "Redis database")
		redisPool = fs.Int("redis.pool_size", 10, "Redis connection pool size")
	)

	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	fs.Parse(os.Args[1:])

	// Log
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	// Metrics
	var counts metrics.Counter
	{
		counts = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "pamao",
			Subsystem: "wechattokensvc",
			Name:      "invoke_count",
			Help:      "Total count of invoke.",
		}, []string{"appid"})
	}
	var duration metrics.Histogram
	{
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "pamao",
			Subsystem: "wechattokensvc",
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds",
		}, []string{"method", "success"})
	}
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())

	// Tracer
	var tracer stdopentracing.Tracer
	{
		cfg := jaegerconfig.Configuration{
			Sampler: &jaegerconfig.SamplerConfig{
				Type:  "const",
				Param: 1,
			},
			Reporter: &jaegerconfig.ReporterConfig{
				LogSpans:            true,
				BufferFlushInterval: time.Second,
			},
		}
		tr, closer, err := cfg.New(
			"wechattokensvc",
			jaegerconfig.Logger(jaeger.StdLogger),
		)
		if err != nil {
			panic(err)
		}
		stdopentracing.SetGlobalTracer(tr)
		defer closer.Close()
		tracer = stdopentracing.GlobalTracer()
	}

	var rc cache.Cache
	{
		rc = cache.NewRedis(&cache.RedisOpts{
			Host:      *redisAddr,
			Password:  *redisPass,
			Database:  *redisDB,
			MaxIdle:   *redisPool,
			MaxActive: *redisPool,
		})
	}

	var (
		svc         = service.New(rc, logger, counts)
		endpoints   = endpoint.New(svc, duration, tracer)
		httpHandler = transport.NewHTTPHandler(endpoints, tracer, logger)
		grpcServer  = transport.NewGRPCServer(endpoints, tracer, logger)
	)

	var g group.Group
	{
		debugListener, err := net.Listen("tcp", *debugAddr)
		if err != nil {
			logger.Log("transport", "debug/HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "debug/HTTP", "addr", *debugAddr)
			return http.Serve(debugListener, http.DefaultServeMux)
		}, func(error) {
			debugListener.Close()
		})
	}
	{
		listener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(listener, httpHandler)
		}, func(error) {
			listener.Close()
		})
	}
	{
		listener, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			logger.Log("transport", "gRPC", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "gRPC", "addr", *grpcAddr)
			baseServer := grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
			pb.RegisterWechatTokenServer(baseServer, grpcServer)
			return baseServer.Serve(listener)
		}, func(error) {
			listener.Close()
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancel:
				return nil
			}
		}, func(error) {
			close(cancel)
		})
	}
	logger.Log("exit", g.Run())
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
