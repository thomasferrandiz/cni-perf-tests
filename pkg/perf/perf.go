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
	TypeNodeService
	TypePodNode
	TypePodPod
	TypeBareMetal
)

var typeName = map[testType]string{
	TypeNodePod:     "node-to-pod",
	TypeNodeService: "node-to-service",
	TypePodNode:     "pod-to-node",
	TypePodPod:      "pod-to-pod",
	TypeBareMetal:   "baremetal",
}

const (
	udpMonoCommand  = " -O 5 -u -b 0 -Z -t 25 --json"
	udpMultiCommand = " -O 5 -u -b 0 -Z -P 16 -t 25 --json"
	udpPPSCommand   = " -O 5 -u -b 0 -Z -t 25 -l 64 --json"
	tcpMonoCommand  = " -t 25 -O 5 -P 1 -Z --dont-fragment --json"
	tcpMultiCommand = " -t 25 -O 5 -P 16 -Z --dont-fragment --json"
)

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
	PPS
)

var streamName = map[streamType]string{
	StreamMono:  "mono-stream",
	StreamMulti: "multi-stream",
	PPS:         "packets-per-second",
}

func (st streamType) String() string {
	return streamName[st]
}

type testResult struct {
	testType   testType
	streamType streamType
	protocol   string
	rates      []float64 // one entry for each run of the test
	losses     []int64   // one entry for each run of the test, UDP: lost packets, TCP: retransmits
	latency    float64
}

type testResults []testResult

const (
	waitForIperf3Server = 3 //time in seconds
	iPerf3ServerCommand = "iperf3 -s -1"
)

func averageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	} else {
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	}
}

func exportToCSV(results testResults) []string {

	nbRate := 0
	for _, result := range results {
		nbRate = max(nbRate, len(result.rates))
	}
	rates_header := ""
	for i := 0; i < nbRate; i++ {
		rates_header = fmt.Sprintf("%srate #%d,", rates_header, i)
	}
	lp_header := ""
	for i := 0; i < nbRate; i++ {
		lp_header = fmt.Sprintf("%slost packets/retransmits #%d,", lp_header, i)
	}

	header_line := fmt.Sprintf("protocol,type,stream type,%savg rate,%s\n", rates_header, lp_header)
	var bres []string = []string{header_line}
	for _, result := range results {
		jrates, _ := json.Marshal(result.rates)
		jlp, _ := json.Marshal(result.losses)
		b := fmt.Sprintf("%s,%s,%s,%s,%f,%s\n", result.protocol, result.testType, result.streamType, strings.Trim(string(jrates), "[]"), averageFloat64(result.rates), strings.Trim(string(jlp), "[]"))
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

	nsRes, err := NodeToServicePerfTests(ctx, masterNode, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run NodeToServicePerfTests tests: %v", err)
	} else {
		results = append(results, nsRes...)
	}

	pnRes, err := PodToNodePerfTests(ctx, masterNode, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run PodToNodePerfTests tests: %v", err)
	} else {
		results = append(results, pnRes...)
	}

	ppRes, err := PodToPodPerfTests(ctx, masterNode, clientHost, serverHost, nbIter, false)
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

func SriovTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) {
	results := make(testResults, 0)
	bmRes, err := BareMetalPerfTests(ctx, clientHost, serverHost, nbIter)
	if err != nil {
		log.Errorf("Couldn't run baremetal tests: %v", err)
	} else {
		results = append(results, bmRes...)
	}

	ppRes, err := PodToPodPerfTests(ctx, masterNode, clientHost, serverHost, nbIter, true)
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
