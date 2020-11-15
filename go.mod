module arhat.dev/pkg

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.15
	github.com/creack/pty v1.1.11
	github.com/itchyny/gojq v0.11.2
	github.com/pion/dtls/v2 v2.0.3
	github.com/pion/udp v0.1.0
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201112155050-0c6587e931a9
	golang.org/x/sys v0.0.0-20201113233024-12cec1faf1ba
	google.golang.org/grpc v1.33.2
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kubernetes v1.19.4
)

require (
	go.opentelemetry.io/otel v0.13.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.13.0
	go.opentelemetry.io/otel/exporters/otlp v0.13.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.13.0
	go.opentelemetry.io/otel/exporters/trace/zipkin v0.13.0
	go.opentelemetry.io/otel/sdk v0.13.0
)

replace (
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/apiserver => k8s.io/apiserver v0.19.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.4
	k8s.io/code-generator => k8s.io/code-generator v0.19.4
	k8s.io/component-base => k8s.io/component-base v0.19.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.19.4
	k8s.io/cri-api => k8s.io/cri-api v0.19.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.4
	k8s.io/kubectl => k8s.io/kubectl v0.19.4
	k8s.io/kubelet => k8s.io/kubelet v0.19.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.4
	k8s.io/metrics => k8s.io/metrics v0.19.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.4
)
