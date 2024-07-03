# cni-perf-tests
Tool to test the performance of k8s CNI plugins

# How to build
```
go build
```

# How to use
* Deploy an rke2 cluster with 1 server and 2 agents
* Apply the manifest `manifests/iperf3.yaml` to your cluster
* Build the code
* Start the tests:
```
./cni-perf-tests runTests
```

# Limitations
* The names and IP addresses of the servers are hard-coded 