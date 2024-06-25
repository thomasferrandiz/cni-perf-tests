package perf

import (
	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

// check that iperf3 is present on local host
func CheckLocalIPerf3() bool {
	res, err := utils.RunCommand("iperf3 --version")

	if err != nil {
		log.Errorf("error with local iperf3: %v", err)
		return false
	}
	log.Infof("Found iperf3: %v", res)
	return true
}

func runIperf3Server(host utils.SshConfig) (string, error) {
	res, err := utils.RunCommandRemotely(host, "iperf3 -s")
	if err != nil {
		return "", err
	}
	return res, nil
}

func runIperf3TcpMono(host utils.SshConfig, serverAddr string) (string, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -t 25 -O 5 -P 1 -Z --dont-fragment"

	res, err := utils.RunCommandRemotely(host, iperf3Command)
	if err != nil {
		return "", err
	}
	return res, nil
}

func BareMetalPerfTests(clientHost, serverHost utils.SshConfig) error {
	go runIperf3Server(serverHost)

	res, err := runIperf3TcpMono(clientHost, serverHost.Hostname)
	if err != nil {
		return err
	}
	log.Infof("test result:\n%s", res)
	return nil
}
