package perf

import (
	"context"
	"sync"
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

func runBMIPerf3Command(host utils.SshConfig, serverAddr, command string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + command

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
	results := make(testResults, 5)

	results[0] = testResult{
		testType:   TypeBareMetal,
		streamType: StreamMono,
		protocol:   TCPProtocol,
	}
	results[1] = testResult{
		testType:   TypeBareMetal,
		streamType: StreamMono,
		protocol:   UDPProtocol,
	}
	results[2] = testResult{
		testType:   TypeBareMetal,
		streamType: StreamMulti,
		protocol:   TCPProtocol,
	}
	results[3] = testResult{
		testType:   TypeBareMetal,
		streamType: StreamMulti,
		protocol:   UDPProtocol,
	}
	results[4] = testResult{
		testType:   TypeBareMetal,
		streamType: PPS,
		protocol:   UDPProtocol,
	}
	//WaitGroup to sync with the iperf3 server goroutine
	wg := sync.WaitGroup{}

	for i := 0; i < nbIter; i++ {
		// res, err := runLatencyTest(clientHost, serverHost.IpAddr)
		// if err != nil {
		// 	return nil, err
		// }
		// latency, err := utils.ParsePingOutput(res)
		// if err != nil {
		// 	return nil, err
		// }
		log.Infof("##### Running baremetal test [ %d ] #####", i)
		// TCP Mono
		log.Infof("    ##### TCP Mono")
		wg.Add(1)

		go func() {
			runBMIperf3Server(ctx, serverHost)
			wg.Done()
		}()
		time.Sleep(waitForIperf3Server * time.Second)

		res, err := runBMIPerf3Command(clientHost, serverHost.TestIpAddr, tcpMonoCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, err := utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[0].rates = append(results[0].rates, tcpRate)
		wg.Wait()

		// TCP Multi
		log.Infof("    ##### TCP Multi")
		wg.Add(1)

		go func() {
			runBMIperf3Server(ctx, serverHost)
			wg.Done()
		}()
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runBMIPerf3Command(clientHost, serverHost.TestIpAddr, tcpMultiCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, err = utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[2].rates = append(results[2].rates, tcpRate)

		wg.Wait()

		//UDP Mono
		log.Infof("    ##### UDP Mono")
		wg.Add(1)

		go func() {
			runBMIperf3Server(ctx, serverHost)
			wg.Done()
		}()
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runBMIPerf3Command(clientHost, serverHost.TestIpAddr, udpMonoCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err := utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[1].rates = append(results[1].rates, udpRate)
		results[1].lost_packets = append(results[1].lost_packets, udpLP)
		wg.Wait()

		//UDP Multi
		log.Infof("    ##### UDP Multi")
		wg.Add(1)

		go func() {
			runBMIperf3Server(ctx, serverHost)
			wg.Done()
		}()
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runBMIPerf3Command(clientHost, serverHost.TestIpAddr, udpMultiCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[3].rates = append(results[3].rates, udpRate)
		results[3].lost_packets = append(results[3].lost_packets, udpLP)
		wg.Wait()

		//UDP PPS
		log.Infof("    ##### UDP PPS")
		wg.Add(1)

		go func() {
			runBMIperf3Server(ctx, serverHost)
			wg.Done()
		}()
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runBMIPerf3Command(clientHost, serverHost.TestIpAddr, udpPPSCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[4].rates = append(results[4].rates, udpRate)
		results[4].lost_packets = append(results[4].lost_packets, udpLP)
		wg.Wait()
	}

	return results, nil
}
