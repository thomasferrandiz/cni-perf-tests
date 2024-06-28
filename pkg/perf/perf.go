package perf

import (
	"context"
	"time"

	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

type testType int

const (
	TypeNodePod = iota
	TypePodNode
	TypePodPod
	TypeBareMetal
)

var typeName = map[testType]string{
	TypeNodePod:   "node-to-pod",
	TypePodNode:   "pod-to-node",
	TypePodPod:    "pod-to-pod",
	TypeBareMetal: "baremetal",
}

func (tt testType) String() string {
	return typeName[tt]
}

const (
	TCPProtocol = "TCP"
	UDPProtocol = "UDP"
)

type testResult struct {
	testType testType
	protocol string
	rate     float64
}

type testResults []testResult

const (
	waitForIperf3Server = 2 //time in seconds
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

func runIperf3Server(ctx context.Context, host utils.SshConfig) {
	log.Infof("Starting remote iperf3 server...")
	_, err := utils.RunCommandRemotely(host, "iperf3 -s -1")
	if err != nil {
		log.Errorf("Error while running iperf3 server: %v", err)
	}
}

func runIperf3TcpMono(host utils.SshConfig, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -t 25 -O 5 -P 1 -Z --dont-fragment --json"
	// iperf3Command := "iperf3 -c " + serverAddr + " -t 5 -P 1 -Z --dont-fragment --json"

	res, err := utils.RunCommandRemotely(host, iperf3Command)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runIPerf3UdpMono(host utils.SshConfig, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -O 5 -u -b 0 -Z -t 25 --json"

	res, err := utils.RunCommandRemotely(host, iperf3Command)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func AllPerfTests(ctx context.Context, clientHost, serverHost utils.SshConfig) {
	results := make(testResults, 0)
	bmRes, err := BareMetalPerfTests(ctx, clientHost, serverHost)
	if err != nil {
		log.Errorf("Couldn't run baremetal tests: %v", err)
	} else {
		results = append(results, bmRes...)
	}
	log.Infof("perf test results: %s, %s, %f", results[0].testType, results[0].protocol, results[0].rate)

}

func BareMetalPerfTests(ctx context.Context, clientHost, serverHost utils.SshConfig) (testResults, error) {
	results := make(testResults, 2)
	go runIperf3Server(ctx, serverHost)
	time.Sleep(waitForIperf3Server * time.Second)

	res, err := runIperf3TcpMono(clientHost, serverHost.Hostname)
	if err != nil {
		return nil, err
	}
	tcpRate, err := utils.ParseIperf3JsonOutput(res)
	if err != nil {
		return nil, err
	}

	results = append(results, testResult{
		testType: TypeBareMetal,
		protocol: TCPProtocol,
		rate:     tcpRate,
	})
	time.Sleep(waitForIperf3Server * time.Second)

	go runIperf3Server(ctx, serverHost)
	time.Sleep(waitForIperf3Server * time.Second)

	res, err = runIPerf3UdpMono(clientHost, serverHost.Hostname)
	if err != nil {
		return nil, err
	}
	udpRate, err := utils.ParseIperf3JsonOutput(res)
	if err != nil {
		return nil, err
	}

	results = append(results, testResult{
		testType: TypeBareMetal,
		protocol: UDPProtocol,
		rate:     udpRate,
	})

	return results, nil
}
