package perf

import (
	"context"

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
