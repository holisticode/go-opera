package cli

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"
	"github.com/spf13/cobra"
)

const (
	appName = "p2ptest"

	rpcsFlag              = "node-rpc-endpoints"
	rpcsFlagShort         = "e"
	sequenceFileFlag      = "sequence-file"
	sequenceFileFlagShort = "s"
)

var (
	rootCmd = &cobra.Command{
		Use:   appName,
		Short: "Run p2p tests",
		Long: `P2P tests check the compliance of a node with the protocol.

As such, they can be used for different implementations of the protocol, in different languages`,
		RunE: run,
	}

	rpcs         []string
	sequenceFile string
)

func init() {
	rootCmd.PersistentFlags().StringArrayVarP(&rpcs, rpcsFlag, rpcsFlagShort, []string{}, "list of node RPC endpoints")
	rootCmd.PersistentFlags().StringVarP(&sequenceFile, sequenceFileFlag, sequenceFileFlagShort, "cmd/p2ptest/suite/default.json", "path to file containing sequence of tests")
}

func run(cmd *cobra.Command, args []string) error {
	seq, err := proto.LoadSequence(sequenceFile)
	if err != nil {
		return nil
	}

	return seq.Run(rpcs)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute %s: '%s'", appName, err)
		os.Exit(1)
	}
}
