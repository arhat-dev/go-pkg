package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"arhat.dev/pkg/iohelper"
)

func log(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

const (
	caCert    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	tokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	caCertDataFrom string
	tokenDataFrom  string
	kubernetesHost string
	kubernetesPort string
	kubeconfig     string
)

func init() {
	flag.StringVar(&caCertDataFrom, "ca-data-from", "", "file to provide kubernetes client ca data")
	flag.StringVar(&tokenDataFrom, "token-data-from", "", "file to provide kubernetes client token data")
	flag.StringVar(&kubernetesHost, "host", "invalid-kubernetes", "kubernetes api host")
	flag.StringVar(&kubernetesPort, "port", "65534", "kubernetes api port")
	flag.StringVar(&kubeconfig, "config", "./test/testdata/kubeconfig.yaml", "path to kubeconfig file")
}

func main() {
	flag.Parse()

	extraCmd := flag.Args()

	if len(extraCmd) == 0 {
		log("must provide extra command")
		os.Exit(1)
	}

	var err error
	kubeconfig, err = filepath.Abs(kubeconfig)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path of kubeconfig: %w", err))
	}

	err = os.Setenv("TEST_KUBECONFIG", kubeconfig)
	if err != nil {
		panic(fmt.Errorf("failed to set kubernetes kubeconfig env: %w", err))
	}

	err = os.Setenv("KUBERNETES_SERVICE_HOST", kubernetesHost)
	if err != nil {
		panic(fmt.Errorf("failed to set kubernetes host env: %w", err))
	}

	err = os.Setenv("KUBERNETES_SERVICE_PORT", kubernetesPort)
	if err != nil {
		panic(fmt.Errorf("failed to set kubernetes port env: %w", err))
	}

	tokenData, err := ioutil.ReadFile(tokenDataFrom)
	if err != nil {
		panic(fmt.Errorf("failed to read token data: %w", err))
	}

	caData, err := ioutil.ReadFile(caCertDataFrom)
	if err != nil {
		panic(fmt.Errorf("failed to read ca data: %w", err))
	}

	undoTokenFile, err := iohelper.WriteFile(tokenFile, tokenData, 0644, false)
	if err != nil {
		panic(fmt.Errorf("failed to ensure token file: %w", err))
	}

	defer func() {
		undoErr := undoTokenFile()
		if undoErr != nil {
			log("failed to undo token file creation: %v", undoErr)
		}
	}()

	undoCaFile, err := iohelper.WriteFile(caCert, caData, 0644, false)
	if err != nil {
		log("failed to ensure ca cert: %v", err)
		return
	}

	defer func() {
		undoErr := undoCaFile()
		if undoErr != nil {
			log("failed to undo token file creation: %v", undoErr)
		}
	}()

	cmd := exec.Command(extraCmd[0], extraCmd[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		log(err.Error())
	}
}
