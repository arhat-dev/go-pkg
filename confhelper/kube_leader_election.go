// +build !nocloud,!nokube

package confhelper

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"arhat.dev/pkg/envhelper"
	"arhat.dev/pkg/log"
)

func FlagsForLeaderElection(name, prefix string, c *LeaderElectionConfig) *pflag.FlagSet {
	fs := pflag.NewFlagSet("kube.leader-election", pflag.ExitOnError)

	fs.StringVar(&c.Identity, prefix+"identity", envhelper.ThisPodName(), "set identity used for leader election")
	// lock
	fs.StringVar(&c.Lock.Type, prefix+"lock.type", "leases", "set resource lock type for leader election, possible values are [configmaps, endpoints, leases, configmapsleases, endpointsleases]")
	fs.StringVar(&c.Lock.Name, prefix+"lock.name", fmt.Sprintf("%s-leader-election", name), "set resource lock name")
	fs.StringVar(&c.Lock.Namespace, prefix+"lock.namespace", envhelper.ThisPodNS(), "set resource lock namespace")
	// lease
	fs.DurationVar(&c.Lease.Expiration, prefix+"lease.expiration", 15*time.Second, "set duration a lease is valid")
	fs.DurationVar(&c.Lease.RetryInterval, prefix+"lease.retryInterval", 1*time.Second, "set intervals between failed lease renew")
	fs.DurationVar(&c.Lease.RenewTimeout, prefix+"lease.renewTimeout", 5*time.Second, "set timeout duration for lease renew")
	fs.DurationVar(&c.Lease.ExpiryToleration, prefix+"lease.expiryToleration", 10*time.Second, "set how long we will wait until try to acquire lease after lease has expired")

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
	_ = evb.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events(envhelper.ThisPodNS())})

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
