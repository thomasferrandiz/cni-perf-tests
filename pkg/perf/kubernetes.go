package perf

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

const (
	kubectl                    = "/var/lib/rancher/rke2/bin/kubectl "
	kubeconfig                 = " --kubeconfig=/etc/rancher/rke2/rke2.yaml --server=https://127.0.0.1:6443 "
	kubeconfigPath             = "/etc/rancher/rke2/rke2.yaml"
	iperf3ServerPodCommand     = kubectl + kubeconfig + "run iperf3server --image networkstatic/iperf3:latest --overrides='{ \"spec\": {  \"nodeName\":\"%s\" } }'-- %s"
	iperf3ServerKillPodCommand = kubectl + kubeconfig + "delete pod iperf3server"
	iperf3ServerIpAddrCommand  = kubectl + kubeconfig + "get pod %s --template={{.status.podIP}} -n iperf3-test"
	iperf3ServerGetCommand     = kubectl + kubeconfig + "get pods --field-selector 'spec.nodeName=%s' -n iperf3-test --no-headers"
	iperf3PodRunCommand        = kubectl + kubeconfig + "-n iperf3-test exec %s -- %s"
)

func startPodIperf3Server(ctx context.Context, masterNode utils.SshConfig, serverName string) {
	log.Infof("Starting remote iperf3 server in pod...")
	_, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServerPodCommand, serverName, iPerf3ServerCommand))
	if err != nil {
		log.Errorf("Error while starting iperf3 server: %v", err)
	}
}

func killPodIperf3Server(ctx context.Context, masterNode utils.SshConfig) {
	log.Infof("Killing remote iperf3 server in pod...")
	_, err := utils.RunCommandRemotely(masterNode, iperf3ServerKillPodCommand)
	if err != nil {
		log.Errorf("Error while killing iperf3 server: %v", err)
	}
}

func getIperf3ServerPodName(masterNode utils.SshConfig, workerNodeName string) (string, error) {
	bres, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServerGetCommand, workerNodeName))
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
	res, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3ServerIpAddrCommand, podName))
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func runPodIperf3TcpMono(masterNode utils.SshConfig, podName, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -t 25 -O 5 -P 1 -Z --dont-fragment --json"
	res, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3PodRunCommand, podName, iperf3Command))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runPodIperf3TcpMulti(masterNode utils.SshConfig, podName, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -t 25 -O 5 -P 16 -Z --dont-fragment --json"
	res, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3PodRunCommand, podName, iperf3Command))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runPodIperf3UdpMono(masterNode utils.SshConfig, podName, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -O 5 -u -b 0 -Z -t 25 --json"
	res, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3PodRunCommand, podName, iperf3Command))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func runPodIperf3UdpMulti(masterNode utils.SshConfig, podName, serverAddr string) ([]byte, error) {
	iperf3Command := "iperf3 -c " + serverAddr + " -O 5 -u -b 0 -Z -P 16 -t 25 --json"
	res, err := utils.RunCommandRemotely(masterNode, fmt.Sprintf(iperf3PodRunCommand, podName, iperf3Command))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func NodeToPodPerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 4)

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

	for i := 0; i < nbIter; i++ {
		podName, err := getIperf3ServerPodName(masterNode, serverHost.Nodename)
		if err != nil {
			return nil, err
		}
		iperf3ServerIpAddr, err := getPodIperf3ServerIpAddr(masterNode, podName)
		if err != nil {
			return nil, fmt.Errorf("couldnt run NodeToPodPerfTests: %v", err)
		}

		//TCP Mono
		res, err := runBMIperf3TcpMono(clientHost, iperf3ServerIpAddr)
		if err != nil {
			return nil, err
		}
		tcpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[0].rates = append(results[0].rates, tcpRate)

		//TCP Multi
		res, err = runBMIperf3TcpMulti(clientHost, iperf3ServerIpAddr)
		if err != nil {
			return nil, err
		}
		tcpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[2].rates = append(results[2].rates, tcpRate)

		//UDP Mono
		res, err = runBMIPerf3UdpMono(clientHost, iperf3ServerIpAddr)
		if err != nil {
			return nil, err
		}
		udpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[1].rates = append(results[1].rates, udpRate)

		//UDP Multi
		res, err = runBMIPerf3UdpMulti(clientHost, iperf3ServerIpAddr)
		if err != nil {
			return nil, err
		}
		udpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}
		results[3].rates = append(results[3].rates, udpRate)

	}

	return results, nil
}

func PodToNodePerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 4)

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

	for i := 0; i < nbIter; i++ {

		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)

		podName, err := getIperf3ServerPodName(masterNode, clientHost.Nodename)
		if err != nil {
			return nil, err
		}

		//TCP Mono
		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)

		res, err := runPodIperf3TcpMono(masterNode, podName, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		tcpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[0].rates = append(results[0].rates, tcpRate)

		//TCP Multi
		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)

		res, err = runPodIperf3TcpMulti(masterNode, podName, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		tcpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[2].rates = append(results[2].rates, tcpRate)

		//UDP Mono
		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)
		res, err = runPodIperf3UdpMono(masterNode, podName, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		udpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[1].rates = append(results[1].rates, udpRate)

		//UDP Multi
		go runBMIperf3Server(ctx, serverHost)
		time.Sleep(waitForIperf3Server * time.Second)
		res, err = runPodIperf3UdpMulti(masterNode, podName, serverHost.TestIpAddr)
		if err != nil {
			return nil, err
		}
		udpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, err
		}

		results[3].rates = append(results[3].rates, udpRate)
		time.Sleep(waitForIperf3Server * time.Second)
	}

	return results, nil
}

func PodToPodPerfTests(ctx context.Context, masterNode, clientHost, serverHost utils.SshConfig, nbIter int) (testResults, error) {
	results := make(testResults, 4)

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

	for i := 0; i < nbIter; i++ {

		serverPodName, err := getIperf3ServerPodName(masterNode, serverHost.Nodename)
		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 server pod name: %w", err)
		}
		iperf3ServerIpAddr, err := getPodIperf3ServerIpAddr(masterNode, serverPodName)
		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 pod address: %w", err)
		}
		clientPodName, err := getIperf3ServerPodName(masterNode, clientHost.Nodename)
		if err != nil {
			return nil, fmt.Errorf("couldn't get iperf3 client pod name")
		}

		//TCP Mono
		res, err := runPodIperf3TcpMono(masterNode, clientPodName, iperf3ServerIpAddr)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3TcpMono: %w ", err)
		}
		tcpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse tcp mono rate: %w", err)
		}
		results[0].rates = append(results[0].rates, tcpRate)

		//TCP Multi
		res, err = runPodIperf3TcpMulti(masterNode, clientPodName, iperf3ServerIpAddr)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3TcpMulti: %w ", err)
		}
		tcpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse tcp multi rate: %w", err)
		}
		results[2].rates = append(results[2].rates, tcpRate)

		//UDP Mono
		res, err = runPodIperf3UdpMono(masterNode, clientPodName, iperf3ServerIpAddr)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpMono: %w ", err)
		}
		udpRate, err := utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse udp mono rate: %w", err)
		}

		results[1].rates = append(results[1].rates, udpRate)

		//UDP Multi
		res, err = runPodIperf3UdpMulti(masterNode, clientPodName, iperf3ServerIpAddr)
		if err != nil {
			return nil, fmt.Errorf("error in runPodIperf3UdpMulti: %w ", err)
		}
		udpRate, err = utils.ParseIperf3JsonOutput(res)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse tcp multi rate: %w", err)
		}

		results[3].rates = append(results[3].rates, udpRate)

	}

	return results, nil
}
