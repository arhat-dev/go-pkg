package kubehelper

import (
	"os"
)

const (
	EnvKeyPodName        = "POD_NAME"
	EnvKeyPodNamespace   = "POD_NAMESPACE"
	EnvKeyPodUID         = "POD_UID"
	EnvKeyWatchNamespace = "WATCH_NAMESPACE"
	EnvKeyJobNamespace   = "JOB_NAMESPACE"
)

var (
	podUID  string
	podName string
	watchNS string
	podNS   string
	jobNS   string
)

func init() {
	var ok bool
	podUID = os.Getenv(EnvKeyPodUID)
	podName = os.Getenv(EnvKeyPodName)

	podNS, ok = os.LookupEnv(EnvKeyPodNamespace)
	if !ok {
		podNS = "default"
	}

	watchNS, ok = os.LookupEnv(EnvKeyWatchNamespace)
	if !ok {
		watchNS = podNS
	}

	jobNS, _ = os.LookupEnv(EnvKeyJobNamespace)
}

func ThisPodUID() string {
	return podUID
}

func ThisPodName() string {
	return podName
}

func WatchNS() string {
	return watchNS
}

func ThisPodNS() string {
	return podNS
}

func JobNS() string {
	return jobNS
}
