/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tferrandiz/cni-perf-tests/pkg/perf"
	"github.com/tferrandiz/cni-perf-tests/pkg/utils"
	log "k8s.io/klog/v2"
)

// runTestsCmd represents the runTests command
var runTestsCmd = &cobra.Command{
	Use:   "runTests",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runTests,
}

// mako1
var masterNode = utils.SshConfig{
	User:     "root",
	KeyPath:  "/Users/tferrandiz/.ssh/id_rsa",
	IpAddr:   "10.84.158.1",
	Port:     22,
	Nodename: "mako1",
}

// mako2
var workerNode1 = utils.SshConfig{
	User:     "root",
	KeyPath:  "/Users/tferrandiz/.ssh/id_rsa",
	IpAddr:   "10.84.158.2",
	Port:     22,
	Nodename: "mako2",
}

// mako3
var workerNode2 = utils.SshConfig{
	User:     "root",
	KeyPath:  "/Users/tferrandiz/.ssh/id_rsa",
	IpAddr:   "10.84.158.3",
	Port:     22,
	Nodename: "mako3",
}

func init() {
	rootCmd.AddCommand(runTestsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runTestsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runTestsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func testConnections() {
	res, err := utils.RunCommandRemotely(masterNode, "kubectl get nodes")
	if err != nil {
		panic(err)
	}
	log.Infof("result on masterNode\n: %v", res)

	res, err = utils.RunCommandRemotely(workerNode1, "iperf3 --version")
	if err != nil {
		panic(err)
	}
	log.Infof("result on workerNode1:\n %v", res)

	res, err = utils.RunCommandRemotely(workerNode2, "iperf3 --version")
	if err != nil {
		panic(err)
	}
	log.Infof("result on workerNode2:\n %v", res)
}

func runTests(cmd *cobra.Command, args []string) {
	fmt.Println("runTests called")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// testConnections()
	perf.AllPerfTests(ctx, masterNode, workerNode1, workerNode2)
}
