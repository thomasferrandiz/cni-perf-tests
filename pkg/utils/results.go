package utils

import (
	"encoding/json"
)

type results struct {
	Start     interface{}
	Intervals interface{}
	End       end
}

type end struct {
	Streams                 interface{}
	Sum_sent                sum_sent
	Sum_received            interface{}
	Cpu_utilization_percent interface{}
	Sender_tcp_congestion   interface{}
	Receiver_tcp_congestion interface{}
}

type sum_sent struct {
	Start, End, Seconds, Bytes, Bits_per_second float64
	Retransmits                                 int64
	Sender                                      bool
}

func ParseIperf3JsonOutput(result []byte) (float64, error) {
	var res results
	err := json.Unmarshal(result, &res)
	if err != nil {
		return -1, err
	}

	return res.End.Sum_sent.Bits_per_second, nil
}
