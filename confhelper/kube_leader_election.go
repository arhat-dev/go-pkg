// +build !nocloud,!nokube

package confhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"arhat.dev/pkg/envhelper"
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

func (c *LeaderElectionConfig) CreateElector(
	name string,
	kubeClient kubernetes.Interface,
	eventRecorder record.EventRecorder,
	onElected func(context.Context),
	onEjected func(),
	onNewLeader func(identity string),
) (*leaderelection.LeaderElector, error) {
	lock, err := resourcelock.New(c.Lock.Type,
		c.Lock.Namespace,
		c.Lock.Name,
		kubeClient.CoreV1(),
		kubeClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      c.Identity,
			EventRecorder: eventRecorder,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create resource lock: %w", err)
	}

	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Name:            name,
		WatchDog:        leaderelection.NewLeaderHealthzAdaptor(c.Lease.ExpiryToleration),
		Lock:            lock,
		LeaseDuration:   c.Lease.Expiration,
		RenewDeadline:   c.Lease.RenewTimeout,
		RetryPeriod:     c.Lease.RetryInterval,
		ReleaseOnCancel: true,

		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onElected,
			OnStoppedLeading: onEjected,
			OnNewLeader:      onNewLeader,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create leader elector: %w", err)
	}

	return elector, nil
}
