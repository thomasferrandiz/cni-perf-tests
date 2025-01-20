package utils

import "testing"

func TestParsePingOutput(t *testing.T) {
	expected_ping := 13.176
	ping_result := []byte(`PING google.fr (172.217.20.163): 56 data bytes
64 bytes from 172.217.20.163: icmp_seq=0 ttl=115 time=11.182 ms
64 bytes from 172.217.20.163: icmp_seq=1 ttl=115 time=14.129 ms
64 bytes from 172.217.20.163: icmp_seq=2 ttl=115 time=14.216 ms

--- google.fr ping statistics ---
3 packets transmitted, 3 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 11.182/13.176/14.216/1.410 ms`)

	ping, err := ParsePingOutput(ping_result)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if ping != expected_ping {
		t.Fatalf("Invalid value for ping. Expected: %f but got %f", expected_ping, ping)
	}
}
