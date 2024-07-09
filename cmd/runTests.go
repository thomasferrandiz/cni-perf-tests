/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"

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

// variables to store the command flag values
// Path to configuration file for the servers
var serversFilename string

// Number of interation for each test
var nbIter int

func init() {
	rootCmd.AddCommand(runTestsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	runTestsCmd.PersistentFlags().StringVar(&serversFilename, "servers", "", "path to yaml file with servers list")
	runTestsCmd.PersistentFlags().IntVar(&nbIter, "iterations", 1, "number of iteration for each test")
	// runTestsCmd.PersistentFlags().String("servers", "", "path to yaml file with servers list")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runTestsCmd.Flags().String("servers", "", "path to yaml file with servers list")
	// viper.BindPFlag("servers", runTestsCmd.PersistentFlags().Lookup("servers"))
}

func runTests(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("Number of iterations: %d", nbIter)
	log.Infof("servers file: %s", serversFilename)
	servers, err := utils.ReadConfigFile(serversFilename)
	if err != nil {
		log.Errorf("could not read servers file: %v", err)
	}
	log.Infof("servers: %v", servers)

	perf.AllPerfTests(ctx, servers.MasterNode, servers.WorkerNode1, servers.WorkerNode2, nbIter)
}
