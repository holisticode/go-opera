package cli

import (
	"fmt"
	"os"

	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/proto"
	"github.com/Fantom-foundation/go-opera/cmd/p2ptest/suite"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	appName = "p2ptest"

	rpcsFlag              = "node-rpc-endpoints"
	rpcsFlagShort         = "e"
	sequenceFileFlag      = "sequence-file"
	sequenceFileFlagShort = "s"
	logLevelFlag          = "log-level"
	logLevelFlagShort     = "l"
)

var (
	rootCmd = &cobra.Command{
		Use:   appName,
		Short: "Run p2p tests",
		Long: `P2P tests check the compliance of a node with the protocol.

As such, they can be used for different implementations of the protocol, in different languages`,
		RunE: run,
	}

	logLevel     string
	sequenceFile string
	rpcs         []string
)

func init() {
	rootCmd.PersistentFlags().StringArrayVarP(&rpcs, rpcsFlag, rpcsFlagShort, []string{}, "list of node RPC endpoints")
	rootCmd.PersistentFlags().StringVarP(&sequenceFile, sequenceFileFlag, sequenceFileFlagShort, "cmd/p2ptest/suite/default.json", "path to file containing sequence of tests")
	rootCmd.PersistentFlags().StringVarP(&logLevel, logLevelFlag, logLevelFlagShort, "info", "log level to use")
}

func run(cmd *cobra.Command, args []string) error {
	var (
		logger *zap.Logger
		err    error
	)

	cfg := zap.NewProductionConfig()
	cfg.Level, err = zap.ParseAtomicLevel(logLevel)
	logger, err = cfg.Build()
	if err != nil {
		return err
	}

	seq, err := proto.LoadSequence(suite.InitialSuite, logger)
	if err != nil {
		return err
	}

	return seq.Run(rpcs)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute %s: '%s'", appName, err)
		os.Exit(1)
	}
}
