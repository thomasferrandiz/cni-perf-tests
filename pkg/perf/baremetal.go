package perf

import (
	"context"
	"time"

	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

func runBMIperf3Server(ctx context.Context, host utils.SshConfig) {
	log.Infof("Starting remote iperf3 server...")
	_, err := utils.RunCommandRemotely(host, iPerf3ServerCommand)
	if err != nil {
		log.Errorf("Error while running iperf3 server: %v", err)
	}
}

func runBMIperf3TcpMono(host utils.SshConfig, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -t 25 -O 5 -P 1 -Z --dont-fragment --json"
	// iperf3Command := "iperf3 -c " + serverAddr + " -t 5 -P 1 -Z --dont-fragment --json"

	res, err := utils.RunCommandRemotely(host, iperf3Command)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runBMIPerf3UdpMono(host utils.SshConfig, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -O 5 -u -b 0 -Z -t 25 --json"

	res, err := utils.RunCommandRemotely(host, iperf3Command)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runLatencyTest(host utils.SshConfig, serverAddr string) ([]byte, error) {
	pingCommand := "ping -c 10 " + serverAddr

	res, err := utils.RunCommandRemotely(host, pingCommand)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func BareMetalPerfTests(ctx context.Context, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 2)

	results[0] = testResult{
		testType: TypeBareMetal,
		protocol: TCPProtocol,
	}
	results[1] = testResult{
		testType: TypeBareMetal,
		protocol: UDPProtocol,
	}
	for i := 0; i < nbIter; i++ {
		// res, err := runLatencyTest(clientHost, serverHost.IpAddr)
		// if err != nil {
		// 	return nil, err
		// }
		// latency, err := utils.ParsePingOutput(res)
		// if err != nil {
		// 	return nil, err
		// }

		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)

		res, err := runBMIperf3TcpMono(clientHost, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		tcpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[0].rates = append(results[0].rates, tcpRate)

		time.Sleep(waitForIperf3Server * time.Second)

		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runBMIPerf3UdpMono(clientHost, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		udpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[1].rates = append(results[1].rates, udpRate)
	}

	return results, nil
}
