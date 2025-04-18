package perf

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

type streamType int

const (
	StreamMono = iota
	StreamMulti
)

var streamName = map[streamType]string{
	StreamMono:  "mono-stream",
	StreamMulti: "multi-stream",
}

func (st streamType) String() string {
	return streamName[st]
}

type testResult struct {
	testType   testType
	streamType streamType
	protocol   string
	rate       float64
	rates      []float64 // one entry for each run of the test
	latency    float64
}

type testResults []testResult

const (
	waitForIperf3Server = 3 //time in seconds
	iPerf3ServerCommand = "iperf3 -s -1"
)

func exportToCSV(results testResults) []string {
	var bres []string
	for _, result := range results {
		jrates, _ := json.Marshal(result.rates)
		b := fmt.Sprintf("%s, %s, %s, %s\n", result.protocol, result.testType, result.streamType, strings.Trim(string(jrates), "[]"))
		bres = append(bres, b)
	}
	return bres
}

func writeCSVFile(results testResults, filename string) error {
	csvResults := exportToCSV(results)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range csvResults {
		_, err := f.WriteString(line)
		if err != nil {
			return err
		}
	}
	f.Sync()
	return nil
}

func AllPerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) {
	results := make(testResults, 0)
	bmRes, err := BareMetalPerfTests(ctx, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run baremetal tests: %v", err)
	} else {
		results = append(results, bmRes...)
	}

	npRes, err := NodeToPodPerfTests(ctx, masterNode, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run NodeToPodPerfTests tests: %v", err)
	} else {
		results = append(results, npRes...)
	}

	pnRes, err := PodToNodePerfTests(ctx, masterNode, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run PodToNodePerfTests tests: %v", err)
	} else {
		results = append(results, pnRes...)
	}

	ppRes, err := PodToPodPerfTests(ctx, masterNode, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run PodToPodPerfTests tests: %v", err)
	} else {
		results = append(results, ppRes...)
	}
	log.Infof("perf test results: %v", results)

	err = writeCSVFile(results, "/tmp/cni-perf-test-results.csv")
	if err != nil {
		log.Errorf("couldn't write results to file: %v", err)
	}
}
