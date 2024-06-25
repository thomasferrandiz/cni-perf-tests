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
