module arhat.dev/pkg

go 1.16

require (
	github.com/Microsoft/go-winio v0.5.0
	github.com/creack/pty v1.1.13
	github.com/itchyny/gojq v0.12.4
	github.com/pion/dtls/v2 v2.0.9
	github.com/pion/udp v0.1.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/sys v0.0.0-20210629170331-7dc0b73dc9fb
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
	go.uber.org/atomic v1.8.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
)

replace github.com/creack/pty => github.com/jeffreystoke/pty v1.1.12-0.20210531091229-b834701fbcc6

replace (
	k8s.io/api => github.com/kubernetes/api v0.20.7
	k8s.io/apiextensions-apiserver => github.com/kubernetes/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery => github.com/kubernetes/apimachinery v0.20.7
	k8s.io/apiserver => github.com/kubernetes/apiserver v0.20.7
	k8s.io/cli-runtime => github.com/kubernetes/cli-runtime v0.20.7
	k8s.io/client-go => github.com/kubernetes/client-go v0.20.7
	k8s.io/cloud-provider => github.com/kubernetes/cloud-provider v0.20.7
	k8s.io/cluster-bootstrap => github.com/kubernetes/cluster-bootstrap v0.20.7
	k8s.io/code-generator => github.com/kubernetes/code-generator v0.20.7
	k8s.io/component-base => github.com/kubernetes/component-base v0.20.7
	k8s.io/component-helpers => github.com/kubernetes/component-helpers v0.20.7
	k8s.io/controller-manager => github.com/kubernetes/controller-manager v0.20.7
	k8s.io/cri-api => github.com/kubernetes/cri-api v0.20.7
	k8s.io/csi-translation-lib => github.com/kubernetes/csi-translation-lib v0.20.7
	k8s.io/kube-aggregator => github.com/kubernetes/kube-aggregator v0.20.7
	k8s.io/kube-controller-manager => github.com/kubernetes/kube-controller-manager v0.20.7
	k8s.io/kube-proxy => github.com/kubernetes/kube-proxy v0.20.7
	k8s.io/kube-scheduler => github.com/kubernetes/kube-scheduler v0.20.7
	k8s.io/kubectl => github.com/kubernetes/kubectl v0.20.7
	k8s.io/kubelet => github.com/kubernetes/kubelet v0.20.7
	k8s.io/legacy-cloud-providers => github.com/kubernetes/legacy-cloud-providers v0.20.7
	k8s.io/metrics => github.com/kubernetes/metrics v0.20.7
	k8s.io/mount-utils => github.com/kubernetes/mount-utils v0.20.7
	k8s.io/sample-apiserver => github.com/kubernetes/sample-apiserver v0.20.7
	k8s.io/sample-cli-plugin => github.com/kubernetes/sample-cli-plugin v0.20.7
	k8s.io/sample-controller => github.com/kubernetes/sample-controller v0.20.7
)
