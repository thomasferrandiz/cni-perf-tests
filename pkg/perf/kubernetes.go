package perf

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tferrandiz/cni-perf-tests/pkg/retry"
	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

// json structure stored in metadata.annotations.k8s\.v1\.cni\.cncf\.io\/network-status
type NetworkConfig struct {
	Name      string         `json:"name"`
	Interface string         `json:"interface,omitempty"`
	IPs       []string       `json:"ips"`
	Default   bool           `json:"default,omitempty"`
	MAC       string         `json:"mac,omitempty"`
	DNS       map[string]any `json:"dns,omitempty"`
}

const (
	kubectl                    = "/var/lib/rancher/rke2/bin/kubectl "
	kubeconfig                 = " --kubeconfig=/etc/rancher/rke2/rke2.yaml --server=https://127.0.0.1:6443 "
	kubeconfigPath             = "/etc/rancher/rke2/rke2.yaml"
	iperf3ServerIpAddrCommand  = kubectl + kubeconfig + "get pod %s --template={{.status.podIP}} -n iperf3-test"
	iperf3ServerGetCommand     = kubectl + kubeconfig + "get pods --field-selector 'spec.nodeName=%s' -n iperf3-test --no-headers"
	iperf3PodRunCommand        = kubectl + kubeconfig + "-n iperf3-test exec %s -- %s"
	iperf3ServiceIpAddrCommand = kubectl + kubeconfig + "get services -n iperf3-test --field-selector 'metadata.name=iperf3-%s' -o=jsonpath='{.items[0].spec.clusterIP}'"
	getMultusPodIpCommand      = kubectl + kubeconfig + `get pods -n iperf3-test -l app=multus-demo --field-selector spec.nodeName=%s -o jsonpath='{.items[0]..metadata.annotations.k8s\.v1\.cni\.cncf\.io\/network-status}'`
	getMultusPodNameCommand    = kubectl + kubeconfig + `get pods -n iperf3-test -l app=multus-demo --field-selector spec.nodeName=%s -o jsonpath='{.items[0].metadata.name}'`
)

func getIperf3ServerPodName(masterNode utils.SshConfig, workerNodeName string) (string, error) {
	bres, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServerGetCommand, workerNodeName))
		},
	)
	if err != nil {
		return "", err
	}
	res := string(bres)
	res = strings.TrimSpace(res)
	split := strings.Split(res, "\n")
	if len(split) == 0 {
		return "", fmt.Errorf("no iperf3 server pod found on node %s", workerNodeName)
	} else if len(split) > 1 {
		return "", fmt.Errorf("too many iperf3 server pods found on node %s", workerNodeName)
	}
	fields := strings.Fields(split[0])
	return fields[0], nil
}

func getPodIperf3ServerIpAddr(masterNode utils.SshConfig, podName string) (string, error) {
	res, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServerIpAddrCommand, podName))
		},
	)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func getPodIperf3ServiceIpAddr(masterNode utils.SshConfig, nodeName string) (string, error) {
	res, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServiceIpAddrCommand, nodeName))
		},
	)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func getMultusPodIpAddr(masterNode utils.SshConfig, nodeName string) (string, error) {
	res, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(getMultusPodIpCommand, nodeName))
		},
	)
	if err != nil {
		return "", err
	}

	var networkStatus []NetworkConfig

	err = json.Unmarshal([]byte(res), &networkStatus)
	if err != nil {
		return "", err
	}

	fmt.Printf("network status: %v\n", networkStatus)

	return networkStatus[1].IPs[0], nil
}

func getMultusPodName(masterNode utils.SshConfig, nodeName string) (string, error) {
	res, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(getMultusPodNameCommand, nodeName))
		},
	)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func runPodIperf3Command(masterNode utils.SshConfig, podName, serverAddr, command string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + command
	res, err := retry.DoWithData(
		func() ([]byte, error) {
			return utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3PodRunCommand, podName, iperf3Command))
		},
	)

	if err != nil {
		return nil, err
	}
	return res, nil
}

func NodeToPodPerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 5)

	results[0] = testResult{
		testType:   TypeNodePod,
		streamType: StreamMono,
		protocol:   TCPProtocol,
	}
	results[1] = testResult{
		testType:   TypeNodePod,
		streamType: StreamMono,
		protocol:   UDPProtocol,
	}
	results[2] = testResult{
		testType:   TypeNodePod,
		streamType: StreamMulti,
		protocol:   TCPProtocol,
	}
	results[3] = testResult{
		testType:   TypeNodePod,
		streamType: StreamMulti,
		protocol:   UDPProtocol,
	}
	results[4] = testResult{
		testType:   TypeNodePod,
		streamType: PPS,
		protocol:   UDPProtocol,
	}
	for i := 0; i < nbIter; i++ {
		log.Infof("##### Running NodetoPod test [ %d ] #####", i)

		podName, err := getIperf3ServerPodName(masterNode, serverHost.Nodename)
		if err != nil {
			return nil, err
		}
		iperf3ServerIpAddr, err := getPodIperf3ServerIpAddr(masterNode, podName)
		if err != nil {
			return nil, fmt.Errorf("couldnt run NodeToPodPerfTests: %v", err)
		}

		//TCP Mono
		log.Infof("    ##### TCP Mono")
		res, err := runBMIPerf3Command(clientHost, iperf3ServerIpAddr, tcpMonoCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err := utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[0].rates = append(results[0].rates, tcpRate)
		results[0].losses = append(results[0].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//TCP Multi
		log.Infof("    ##### TCP Multi")
		res, err = runBMIPerf3Command(clientHost, iperf3ServerIpAddr, tcpMultiCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err = utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[2].rates = append(results[2].rates, tcpRate)
		results[2].losses = append(results[2].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Mono
		log.Infof("    ##### UDP Mono")
		res, err = runBMIPerf3Command(clientHost, iperf3ServerIpAddr, udpMonoCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err := utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[1].rates = append(results[1].rates, udpRate)
		results[1].losses = append(results[1].losses, udpLP)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Multi
		log.Infof("    ##### UDP Multi")
		res, err = runBMIPerf3Command(clientHost, iperf3ServerIpAddr, udpMultiCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[3].rates = append(results[3].rates, udpRate)
		results[3].losses = append(results[3].losses, udpLP)

		time.Sleep(waitForIperf3Server * time.Second)

		//UDP PPS
		log.Infof("    ##### UDP PPS")
		res, err = runBMIPerf3Command(clientHost, iperf3ServerIpAddr, udpPPSCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpPPS: %w ", err)
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse UDP PPS result : %w", err)
		}

		results[4].rates = append(results[4].rates, udpRate)
		results[4].losses = append(results[4].losses, udpLP)

		time.Sleep(waitForIperf3Server * time.Second)
	}

	return results, nil
}

func NodeToServicePerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 5)

	results[0] = testResult{
		testType:   TypeNodeService,
		streamType: StreamMono,
		protocol:   TCPProtocol,
	}
	results[1] = testResult{
		testType:   TypeNodeService,
		streamType: StreamMono,
		protocol:   UDPProtocol,
	}
	results[2] = testResult{
		testType:   TypeNodeService,
		streamType: StreamMulti,
		protocol:   TCPProtocol,
	}
	results[3] = testResult{
		testType:   TypeNodeService,
		streamType: StreamMulti,
		protocol:   UDPProtocol,
	}
	results[4] = testResult{
		testType:   TypeNodeService,
		streamType: PPS,
		protocol:   UDPProtocol,
	}
	for i := 0; i < nbIter; i++ {
		log.Infof("##### Running NodetoService test [ %d ] #####", i)

		iperf3ServiceIpAddr, err := getPodIperf3ServiceIpAddr(masterNode, serverHost.Nodename)
		if err != nil {
			return nil, fmt.Errorf("couldnt run NodeToPodPerfTests: %v", err)
		}

		//TCP Mono
		log.Infof("    ##### TCP Mono")
		res, err := runBMIPerf3Command(clientHost, iperf3ServiceIpAddr, tcpMonoCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err := utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[0].rates = append(results[0].rates, tcpRate)
		results[0].losses = append(results[0].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//TCP Multi
		log.Infof("    ##### TCP Multi")
		res, err = runBMIPerf3Command(clientHost, iperf3ServiceIpAddr, tcpMultiCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err = utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[2].rates = append(results[2].rates, tcpRate)
		results[2].losses = append(results[2].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Mono
		log.Infof("    ##### UDP Mono")
		res, err = runBMIPerf3Command(clientHost, iperf3ServiceIpAddr, udpMonoCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err := utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[1].rates = append(results[1].rates, udpRate)
		results[1].losses = append(results[1].losses, udpLP)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Multi
		log.Infof("    ##### UDP Multi")
		res, err = runBMIPerf3Command(clientHost, iperf3ServiceIpAddr, udpMultiCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[3].rates = append(results[3].rates, udpRate)
		results[3].losses = append(results[3].losses, udpLP)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Multi
		log.Infof("    ##### UDP Multi")
		res, err = runBMIPerf3Command(clientHost, iperf3ServiceIpAddr, udpPPSCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[4].rates = append(results[4].rates, udpRate)
		results[4].losses = append(results[4].losses, udpLP)
		time.Sleep(waitForIperf3Server * time.Second)
	}

	return results, nil
}

func PodToNodePerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 5)

	results[0] = testResult{
		testType:   TypePodNode,
		streamType: StreamMono,
		protocol:   TCPProtocol,
	}
	results[1] = testResult{
		testType:   TypePodNode,
		streamType: StreamMono,
		protocol:   UDPProtocol,
	}
	results[2] = testResult{
		testType:   TypePodNode,
		streamType: StreamMulti,
		protocol:   TCPProtocol,
	}
	results[3] = testResult{
		testType:   TypePodNode,
		streamType: StreamMulti,
		protocol:   UDPProtocol,
	}
	results[4] = testResult{
		testType:   TypePodNode,
		streamType: PPS,
		protocol:   UDPProtocol,
	}
	// WaitGroup to sync with the iperf3 server goroutine
	wg := sync.WaitGroup{}

	for i := 0; i < nbIter; i++ {
		log.Infof("##### Running PodtoNode test [ %d ] #####", i)

		podName, err := getIperf3ServerPodName(masterNode, clientHost.Nodename)
		if err != nil {
			return nil, err
		}

		//TCP Mono
		log.Infof("    ##### TCP Mono")

		wg.Go(func() {
			runBMIperf3Server(ctx, serverHost)
		})
		time.Sleep(waitForIperf3Server * time.Second)

		res, err := runPodIperf3Command(masterNode, podName, serverHost.TestIpAddr, tcpMonoCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err := utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[0].rates = append(results[0].rates, tcpRate)
		results[0].losses = append(results[0].losses, retransmits)
		wg.Wait()

		//TCP Multi
		log.Infof("    ##### TCP Multi")

		wg.Go(func() {
			runBMIperf3Server(ctx, serverHost)
		})
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runPodIperf3Command(masterNode, podName, serverHost.TestIpAddr, tcpMultiCommand)
		if err != nil {
			return nil, err
		}
		tcpRate, retransmits, err = utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[2].rates = append(results[2].rates, tcpRate)
		results[2].losses = append(results[2].losses, retransmits)
		wg.Wait()

		//UDP Mono
		log.Infof("    ##### UDP Mono")

		wg.Go(func() {
			runBMIperf3Server(ctx, serverHost)
		})
		time.Sleep(waitForIperf3Server * time.Second)
		res, err = runPodIperf3Command(masterNode, podName, serverHost.TestIpAddr, udpMonoCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err := utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[1].rates = append(results[1].rates, udpRate)
		results[1].losses = append(results[1].losses, udpLP)
		wg.Wait()

		//UDP Multi
		log.Infof("    ##### UDP Multi")

		wg.Go(func() {
			runBMIperf3Server(ctx, serverHost)
		})
		time.Sleep(waitForIperf3Server * time.Second)
		res, err = runPodIperf3Command(masterNode, podName, serverHost.TestIpAddr, udpMultiCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[3].rates = append(results[3].rates, udpRate)
		results[3].losses = append(results[3].losses, udpLP)
		wg.Wait()

		//UDP Multi
		log.Infof("    ##### UDP PPS")

		wg.Go(func() {
			runBMIperf3Server(ctx, serverHost)
		})
		time.Sleep(waitForIperf3Server * time.Second)
		res, err = runPodIperf3Command(masterNode, podName, serverHost.TestIpAddr, udpPPSCommand)
		if err != nil {
			return nil, err
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[4].rates = append(results[4].rates, udpRate)
		results[4].losses = append(results[4].losses, udpLP)
		wg.Wait()
	}

	return results, nil
}

func PodToPodPerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int, useSriov bool) (testResults, error) {
	results := make(testResults, 5)

	results[0] = testResult{
		testType:   TypePodPod,
		streamType: StreamMono,
		protocol:   TCPProtocol,
	}
	results[1] = testResult{
		testType:   TypePodPod,
		streamType: StreamMono,
		protocol:   UDPProtocol,
	}
	results[2] = testResult{
		testType:   TypePodPod,
		streamType: StreamMulti,
		protocol:   TCPProtocol,
	}
	results[3] = testResult{
		testType:   TypePodPod,
		streamType: StreamMulti,
		protocol:   UDPProtocol,
	}
	results[4] = testResult{
		testType:   TypePodPod,
		streamType: PPS,
		protocol:   UDPProtocol,
	}

	for i := 0; i < nbIter; i++ {
		log.Infof("##### Running PodToPod test [ %d ] #####", i)
		var iperf3ServerIpAddr string
		serverPodName, err := getIperf3ServerPodName(masterNode, serverHost.Nodename)
		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 server pod name: %w", err)
		}
		if useSriov {
			iperf3ServerIpAddr, err = getMultusPodIpAddr(masterNode, serverHost.Nodename)
		} else {
			iperf3ServerIpAddr, err = getPodIperf3ServerIpAddr(masterNode, serverPodName)

		}

		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 pod address: %w", err)
		}
		clientPodName, err := getIperf3ServerPodName(masterNode, clientHost.Nodename)
		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 client pod name")
		}

		//TCP Mono
		log.Infof("    ##### TCP Mono")
		res, err := runPodIperf3Command(masterNode, clientPodName, iperf3ServerIpAddr, tcpMonoCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3TcpMono: %w ", err)
		}
		tcpRate, retransmits, err := utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse tcp mono rate: %w", err)
		}
		results[0].rates = append(results[0].rates, tcpRate)
		results[0].losses = append(results[0].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//TCP Multi
		log.Infof("    ##### TCP Multi")
		res, err = runPodIperf3Command(masterNode, clientPodName, iperf3ServerIpAddr, tcpMultiCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3TcpMulti: %w ", err)
		}
		tcpRate, retransmits, err = utils.ParseIperf3TCPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse tcp multi rate: %w", err)
		}
		results[2].rates = append(results[2].rates, tcpRate)
		results[2].losses = append(results[2].losses, retransmits)
		time.Sleep(waitForIperf3Server * time.Second)

		//UDP Mono
		log.Infof("    ##### UDP Mono")
		res, err = runPodIperf3Command(masterNode, clientPodName, iperf3ServerIpAddr, udpMonoCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpMono: %w ", err)
		}
		udpRate, udpLP, err := utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse udp mono rate: %w", err)
		}

		results[1].rates = append(results[1].rates, udpRate)
		results[1].losses = append(results[1].losses, udpLP)

		time.Sleep(waitForIperf3Server * time.Second)
		//UDP Multi
		log.Infof("    ##### UDP Multi")
		res, err = runPodIperf3Command(masterNode, clientPodName, iperf3ServerIpAddr, udpMultiCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpMulti: %w ", err)
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse udp multi rate: %w", err)
		}

		results[3].rates = append(results[3].rates, udpRate)
		results[3].losses = append(results[3].losses, udpLP)

		time.Sleep(waitForIperf3Server * time.Second)

		//UDP PPS
		log.Infof("    ##### UDP PPS")
		res, err = runPodIperf3Command(masterNode, clientPodName, iperf3ServerIpAddr, udpPPSCommand)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpPPS: %w ", err)
		}
		udpRate, udpLP, err = utils.ParseIperf3UDPJsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse UDP PPS result : %w", err)
		}

		results[4].rates = append(results[4].rates, udpRate)
		results[4].losses = append(results[4].losses, udpLP)

		time.Sleep(waitForIperf3Server * time.Second)
	}

	return results, nil
}
