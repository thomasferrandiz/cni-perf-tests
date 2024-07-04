# cni-perf-tests
Tool to test the performance of k8s CNI plugins

# How to build
```
go build
```

# How to use
* Deploy an rke2 cluster with 1 server and 2 agents
* Apply the manifest `manifests/iperf3.yaml` to your cluster
* make sure that you have setup ssh connectivity to all servers
* write a server configuration file like this one:
```yaml
masternode:
  user: myuser
  keypath:  "path/to/ssh/key"
  ipaddr:   "10.0.01"
  port:     22
  nodename: "server1"
workernode1:
  user:     myuser
  keypath:  "path/to/ssh/key"
  ipaddr:   "10.0.0.2"
  port:     22
  nodename: "server2"
workernode2:
  user:     myuser
  keypath:  "path/to/ssh/key"
  ipaddr:   "10.0.0.3"
  port:     22
  nodename: "server3"

```
* Build the code
* Start the tests:
```
./cni-perf-tests runTests --servers servers.yaml
```
* The results of the tests are available in the file /tmp/cni-perf-test-results.csv

