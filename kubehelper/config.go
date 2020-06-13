package kubehelper

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	otapiglobal "go.opentelemetry.io/otel/api/global"
	otapimetric "go.opentelemetry.io/otel/api/metric"
	otapitrace "go.opentelemetry.io/otel/api/trace"
	otprom "go.opentelemetry.io/otel/exporters/metric/prometheus"
	otexporterotlp "go.opentelemetry.io/otel/exporters/otlp"
	otexporterjaeger "go.opentelemetry.io/otel/exporters/trace/jaeger"
	otexporterzipkin "go.opentelemetry.io/otel/exporters/trace/zipkin"
	otsdkmetricspull "go.opentelemetry.io/otel/sdk/metric/controller/pull"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	otsdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/flowcontrol"

	"arhat.dev/pkg/confhelper"
	"arhat.dev/pkg/log"
)

func FlagsForCommonControllerConfig(name, prefix string, l *log.Config, c *CommonControllerConfig) *pflag.FlagSet {
	fs := pflag.NewFlagSet("controller", pflag.ExitOnError)

	// logging
	fs.AddFlagSet(log.FlagsForLogConfig("log.", l))

	// kube client
	fs.AddFlagSet(FlagsForKubeClient(prefix, &c.KubeClient))

	// metrics
	fs.BoolVar(&c.Metrics.Enabled, prefix+"metrics.enabled", true, "enable metrics collection")
	fs.StringVar(&c.Metrics.Endpoint, prefix+"metrics.listen", ":9876", "set address:port for telemetry endpoint")
	fs.StringVar(&c.Metrics.HttpPath, prefix+"metrics.httpPath", "/metrics", "set http path for metrics collection")
	fs.StringVar(&c.Metrics.Format, prefix+"metrics.format", "prometheus", "set metrics format")
	fs.AddFlagSet(confhelper.FlagsForTLSConfig(prefix+"metrics.tls", &c.Metrics.TLS))

	// tracing
	fs.BoolVar(&c.Tracing.Enabled, prefix+"tracing.enabled", false, "enable tracing")
	fs.StringVar(&c.Tracing.Format, prefix+"tracing.format", "jaeger", "set tracing stats format")
	fs.StringVar(&c.Tracing.EndpointType, prefix+"tracing.endpointType", "agent", "set endpoint type of collector (only used for jaeger)")
	fs.StringVar(&c.Tracing.Endpoint, prefix+"tracing.endpoint", "", "set collector endpoint for tracing stats collection")
	fs.Float64Var(&c.Tracing.SampleRate, prefix+"tracing.sampleRate", 1.0, "set tracing sample rate")
	fs.StringVar(&c.Tracing.ReportedServiceName, prefix+"tracing.reportedServiceName", fmt.Sprintf("%s.%s", ThisPodName(), ThisPodNS()), "set service name used for tracing stats")
	fs.AddFlagSet(confhelper.FlagsForTLSConfig(prefix+"tracing.tls", &c.Tracing.TLS))

	// leader election
	fs.StringVar(&c.LeaderElection.Identity, prefix+"leaderElection.identity", ThisPodName(), "set identity used for leader election")
	// lock
	fs.StringVar(&c.LeaderElection.Lock.Type, prefix+"leaderElection.lock.type", "leases", "set resource lock type for leader election, possible values are [configmaps, endpoints, leases, configmapsleases, endpointsleases]")
	fs.StringVar(&c.LeaderElection.Lock.Name, prefix+"leaderElection.lock.name", fmt.Sprintf("%s-leader-election", name), "set resource lock name")
	fs.StringVar(&c.LeaderElection.Lock.Namespace, prefix+"leaderElection.lock.namespace", ThisPodNS(), "set resource lock namespace")
	// lease
	fs.DurationVar(&c.LeaderElection.Lease.Expiration, prefix+"leaderElection.lease.expiration", 15*time.Second, "set duration a lease is valid")
	fs.DurationVar(&c.LeaderElection.Lease.RetryInterval, prefix+"leaderElection.lease.retryInterval", 1*time.Second, "set intervals between failed lease renew")
	fs.DurationVar(&c.LeaderElection.Lease.RenewTimeout, prefix+"leaderElection.lease.renewTimeout", 5*time.Second, "set timeout duration for lease renew")
	fs.DurationVar(&c.LeaderElection.Lease.ExpiryToleration, prefix+"leaderElection.lease.expiryToleration", 10*time.Second, "set how long we will wait until try to acquire lease after lease has expired")

	return fs
}

func FlagsForKubeClient(prefix string, c *KubeClientConfig) *pflag.FlagSet {
	fs := pflag.NewFlagSet("kubeClient", pflag.ExitOnError)

	fs.StringVar(&c.KubeconfigPath, prefix+"kubeClient.kubeconfig", "", "set path to kubeconfig file")

	fs.BoolVar(&c.RateLimit.Enabled, prefix+"kubeClient.rateLimit.enable", true, "enable rate limit for kubernetes api client")
	fs.Float32Var(&c.RateLimit.QPS, prefix+"kubeClient.rateLimit.qps", 5, "set requests per second limit")
	fs.IntVar(&c.RateLimit.Burst, prefix+"kubeClient.rateLimit.burst", 10, "set burst requests per second")

	return fs
}

type LeaderElectionConfig struct {
	Identity string `json:"identity" yaml:"identity"`

	Lease struct {
		Expiration       time.Duration `json:"expiration" yaml:"expiration"`
		RenewTimeout     time.Duration `json:"renewTimeout" yaml:"renewTimeout"`
		RetryInterval    time.Duration `json:"retryInterval" yaml:"retryInterval"`
		ExpiryToleration time.Duration `json:"expiryToleration" yaml:"expiryToleration"`
	} `json:"lease" yaml:"lease"`

	Lock struct {
		Name      string `json:"name" yaml:"name"`
		Namespace string `json:"namespace" yaml:"namespace"`
		Type      string `json:"type" yaml:"type"`
	} `json:"lock" yaml:"lock"`
}

func (c *LeaderElectionConfig) RunOrDie(appCtx context.Context, name string, logger log.Interface, kubeClient kubernetes.Interface, onElected func(context.Context), onEjected func()) {
	// become the leader before proceeding
	evb := record.NewBroadcaster()
	_ = evb.StartLogging(func(format string, args ...interface{}) {
		logger.I(fmt.Sprintf(format, args...), log.String("source", "event"))
	})
	_ = evb.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events(ThisPodNS())})

	rl, err := resourcelock.New(c.Lock.Type,
		c.Lock.Namespace,
		c.Lock.Name,
		kubeClient.CoreV1(),
		kubeClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      c.Identity,
			EventRecorder: evb.NewRecorder(scheme.Scheme, corev1.EventSource{Component: name}),
		})

	if err != nil {
		logger.E("failed to create leader-election lock", log.Error(err))
		os.Exit(1)
	}

	leaderelection.RunOrDie(appCtx, leaderelection.LeaderElectionConfig{
		Name:            name,
		WatchDog:        leaderelection.NewLeaderHealthzAdaptor(c.Lease.ExpiryToleration),
		Lock:            rl,
		LeaseDuration:   c.Lease.Expiration,
		RenewDeadline:   c.Lease.RenewTimeout,
		RetryPeriod:     c.Lease.RetryInterval,
		ReleaseOnCancel: false,

		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onElected,
			OnStoppedLeading: onEjected,
		},
	})
}

type ControllerMetricsConfig struct {
	// Enabled metrics collection
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Endpoint address for metrics/tracing collection,
	// for prometheus, it's a listen address (SHOULD NOT be empty or use random port (:0))
	// for otlp, it's the otlp collector address
	Endpoint string `json:"listen" yaml:"listen"`

	// Format of exposed metrics
	Format string `json:"format" yaml:"format"`

	// HttpPath for metrics collection
	HttpPath string `json:"httpPath" yaml:"httpPath"`

	// TLS config for client/server
	TLS confhelper.TLSConfig `json:"tls" yaml:"tls"`
}

func (c *ControllerMetricsConfig) RegisterIfEnabled(ctx context.Context, logger log.Interface) (err error) {
	if !c.Enabled {
		return nil
	}

	var (
		metricsProvider otapimetric.Provider
	)

	tlsConfig, err := c.TLS.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create tls config: %w", err)
	}

	switch c.Format {
	case "otlp":
		opts := []otexporterotlp.ExporterOption{
			otexporterotlp.WithAddress(c.Endpoint),
		}

		if tlsConfig != nil {
			opts = append(opts, otexporterotlp.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
		} else {
			opts = append(opts, otexporterotlp.WithInsecure())
		}

		var exporter *otexporterotlp.Exporter
		exporter, err = otexporterotlp.NewExporter(opts...)
		if err != nil {
			return fmt.Errorf("failed to create otlp exporter: %w", err)
		}

		pusher := push.New(
			simple.NewWithExactDistribution(),
			exporter,
			push.WithStateful(true),
			push.WithPeriod(5*time.Second),
		)
		pusher.Start()

		metricsProvider = pusher.Provider()
	case "prometheus":
		var metricsListener net.Listener
		metricsListener, err = net.Listen("tcp", c.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to create listener for metrics: %w", err)
		}

		defer func() {
			if err != nil {
				_ = metricsListener.Close()
			}
		}()

		promCfg := otprom.Config{Registry: prom.NewRegistry()}

		var exporter *otprom.Exporter
		exporter, err = otprom.NewExportPipeline(promCfg,
			otsdkmetricspull.WithStateful(true),
			otsdkmetricspull.WithCachePeriod(5*time.Second),
			otsdkmetricspull.WithErrorHandler(func(err error) {
				logger.V("prom push controller error", log.Error(err))
			}),
		)
		if err != nil {
			return fmt.Errorf("failed to install global metrics collector")
		}

		mux := http.NewServeMux()
		mux.Handle(c.HttpPath, exporter)

		if tlsConfig != nil {
			tlsConfig.NextProtos = []string{"h2", "http/1.1"}
		}

		srv := http.Server{
			Handler:   mux,
			TLSConfig: tlsConfig,
			BaseContext: func(net.Listener) context.Context {
				return ctx
			},
		}

		go func() {
			var err error
			defer func() {
				_ = srv.Close()
				_ = metricsListener.Close()

				if err != nil {
					os.Exit(1)
				}
			}()

			if err = srv.Serve(metricsListener); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					err = nil
				} else {
					logger.E("failed to serve metrics", log.Error(err))
				}
			}
		}()
	default:
		return fmt.Errorf("unsupported metrics format %q", c.Format)
	}

	otapiglobal.SetMeterProvider(metricsProvider)

	return nil
}

type ControllerTracingConfig struct {
	// Enabled tracing stats
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Format of exposed tracing stats
	Format string `json:"format" yaml:"format"`

	// EndpointType the type of collector (used for jaeger), can be one of [agent, collector]
	EndpointType string `json:"endpointType" yaml:"endpointType"`

	// Endpoint to report tracing stats
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// SampleRate
	SampleRate float64 `json:"sampleRate" yaml:"sampleRate"`

	// ReportedServiceName used when reporting tracing stats
	ReportedServiceName string `json:"serviceName" yaml:"serviceName"`

	// TLS config for client/server
	TLS confhelper.TLSConfig `json:"tls" yaml:"tls"`
}

func (c *ControllerTracingConfig) newHttpClient(tlsConfig *tls.Config) *http.Client {
	if tlsConfig != nil {
		tlsConfig.NextProtos = []string{"h2", "http/1.1"}
	}

	// TODO: set reasonable defaults, currently using default client and transport
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:       30 * time.Second,
				KeepAlive:     30 * time.Second,
				FallbackDelay: 300 * time.Millisecond,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,

			DialTLS:                nil,
			TLSClientConfig:        tlsConfig,
			DisableKeepAlives:      false,
			DisableCompression:     false,
			MaxIdleConnsPerHost:    0,
			MaxConnsPerHost:        0,
			ResponseHeaderTimeout:  0,
			TLSNextProto:           nil,
			ProxyConnectHeader:     nil,
			MaxResponseHeaderBytes: 0,
			WriteBufferSize:        0,
			ReadBufferSize:         0,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
	}
}

func (c *ControllerTracingConfig) RegisterIfEnabled(ctx context.Context, logger log.Interface) (err error) {
	if !c.Enabled {
		return nil
	}

	var traceProvider otapitrace.Provider

	tlsConfig, err := c.TLS.GetTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create tls config: %w", err)
	}

	switch c.Format {
	case "otlp":
		opts := []otexporterotlp.ExporterOption{
			otexporterotlp.WithAddress(c.Endpoint),
		}

		if tlsConfig != nil {
			opts = append(opts, otexporterotlp.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
		} else {
			opts = append(opts, otexporterotlp.WithInsecure())
		}

		var exporter *otexporterotlp.Exporter
		exporter, err = otexporterotlp.NewExporter(opts...)
		if err != nil {
			return fmt.Errorf("failed to create otlp exporter: %w", err)
		}

		traceProvider, err = otsdktrace.NewProvider(
			otsdktrace.WithConfig(otsdktrace.Config{DefaultSampler: otsdktrace.ProbabilitySampler(c.SampleRate)}),
			otsdktrace.WithSyncer(exporter),
		)
		if err != nil {
			return fmt.Errorf("failed to create trace provider for otlp exporter: %w", err)
		}
	case "zipkin":
		var exporter *otexporterzipkin.Exporter

		exporter, err = otexporterzipkin.NewExporter(c.Endpoint, c.ReportedServiceName,
			otexporterzipkin.WithClient(c.newHttpClient(tlsConfig)),
			otexporterzipkin.WithLogger(nil),
		)
		if err != nil {
			return fmt.Errorf("failed to create zipkin exporter: %w", err)
		}

		traceProvider, err = otsdktrace.NewProvider(
			otsdktrace.WithBatcher(exporter,
				otsdktrace.WithBatchTimeout(5*time.Second),
			),
			otsdktrace.WithConfig(otsdktrace.Config{
				DefaultSampler: otsdktrace.ProbabilitySampler(c.SampleRate),
			}),
		)
		if err != nil {
			return fmt.Errorf("failed to create trace provider for zipkin exporter: %w", err)
		}
	case "jaeger":
		var endpoint otexporterjaeger.EndpointOption
		switch c.EndpointType {
		case "agent":
			endpoint = otexporterjaeger.WithAgentEndpoint(c.Endpoint)
		case "collector":
			otexporterjaeger.WithCollectorEndpoint(c.Endpoint,
				otexporterjaeger.WithUsername(os.Getenv("JAEGER_COLLECTOR_USERNAME")),
				otexporterjaeger.WithPassword(os.Getenv("JAEGER_COLLECTOR_PASSWORD")),
				otexporterjaeger.WithHTTPClient(c.newHttpClient(tlsConfig)),
			)
		default:
			return fmt.Errorf("unsupported tracing endpoint type %q", c.EndpointType)
		}

		var flush func()
		traceProvider, flush, err = otexporterjaeger.NewExportPipeline(endpoint,
			otexporterjaeger.WithProcess(otexporterjaeger.Process{
				ServiceName: c.ReportedServiceName,
			}),
			otexporterjaeger.WithSDK(&otsdktrace.Config{
				DefaultSampler: otsdktrace.ProbabilitySampler(c.SampleRate),
			}),
		)
		_ = flush
	default:
		return fmt.Errorf("unsupported tracing format %q", c.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to create %q tracing provider: %w", c.Format, err)
	}

	otapiglobal.SetTraceProvider(traceProvider)

	return nil
}

type KubeClientConfig struct {
	KubeconfigPath string `json:"kubeconfig" yaml:"kubeconfig"`
	RateLimit      struct {
		Enabled bool    `json:"enabled" yaml:"enabled"`
		QPS     float32 `json:"qps" yaml:"qps"`
		Burst   int     `json:"burst" yaml:"burst"`
	} `json:"rateLimit" yaml:"rateLimit"`
}

// NewKubeClient creates a kubernetes client with/without existing kubeconfig
// if nil kubeconfig was provided, then will retrieve kubeconfig from configured
// path and will fallback to in cluster kubeconfig
// you can choose whether rate limit config is applied, if not, will use default
// rate limit config
func (c *KubeClientConfig) NewKubeClient(kubeconfig *rest.Config, applyRateLimitConfig bool) (client kubernetes.Interface, _ *rest.Config, err error) {
	if kubeconfig != nil {
		kubeconfig = rest.CopyConfig(kubeconfig)
	} else {
		if c.KubeconfigPath != "" {
			kubeconfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: c.KubeconfigPath},
				&clientcmd.ConfigOverrides{}).ClientConfig()

			if err != nil {
				return nil, nil, fmt.Errorf("failed to load kubeconfig from file %q: %w", c.KubeconfigPath, err)
			}
		}

		// fallback to in cluster config
		if kubeconfig == nil {
			kubeconfig, err = rest.InClusterConfig()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load in cluster kubeconfig: %w", err)
			}
		}
	}

	if applyRateLimitConfig {
		if c.RateLimit.Enabled {
			kubeconfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(c.RateLimit.QPS, c.RateLimit.Burst)
		} else {
			kubeconfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
		}
	}

	client, err = kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kube client from kubeconfig: %w", err)
	}

	return client, kubeconfig, nil
}

type CommonControllerConfig struct {
	Log            log.ConfigSet           `json:"log" yaml:"log"`
	KubeClient     KubeClientConfig        `json:"kubeClient" yaml:"kubeClient"`
	Metrics        ControllerMetricsConfig `json:"metrics" yaml:"metrics"`
	Tracing        ControllerTracingConfig `json:"tracing" yaml:"tracing"`
	LeaderElection LeaderElectionConfig    `json:"leaderElection" yaml:"leaderElection"`
}
