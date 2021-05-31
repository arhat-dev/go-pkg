module arhat.dev/pkg

go 1.16

require (
	github.com/Microsoft/go-winio v0.5.0
	github.com/creack/pty v1.1.11
	github.com/itchyny/gojq v0.12.3
	github.com/pion/dtls/v2 v2.0.9
	github.com/pion/udp v0.1.1
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea
	google.golang.org/grpc v1.37.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/kubernetes v1.20.7
)

require (
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0
	go.opentelemetry.io/otel/exporters/otlp v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/exporters/trace/zipkin v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
)

replace github.com/creack/pty => github.com/jeffreystoke/pty v1.1.12-0.20210531091229-b834701fbcc6

replace (
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/apiserver => k8s.io/apiserver v0.20.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.7
	k8s.io/code-generator => k8s.io/code-generator v0.20.7
	k8s.io/component-base => k8s.io/component-base v0.20.7
	k8s.io/component-helpers => k8s.io/component-helpers v0.20.7
	k8s.io/controller-manager => k8s.io/controller-manager v0.20.7
	k8s.io/cri-api => k8s.io/cri-api v0.20.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.7
	k8s.io/kubectl => k8s.io/kubectl v0.20.7
	k8s.io/kubelet => k8s.io/kubelet v0.20.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.7
	k8s.io/metrics => k8s.io/metrics v0.20.7
	k8s.io/mount-utils => k8s.io/mount-utils v0.20.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.7
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.7
	k8s.io/sample-controller => k8s.io/sample-controller v0.20.7
)
