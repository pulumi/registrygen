package cmd

import (
	"github.com/golang/glog"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/logging"
	"github.com/pulumi/registrygen/cmd/docs"
	"github.com/pulumi/registrygen/cmd/metadata"
	"github.com/spf13/cobra"
)

var (
	logToStderr bool
	verbose     int
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "registrygen",
		Short: "Generate Package Metadata and API Docs for the Pulumi registry",
		Long: "A tool to generate API docs and package metadata for Pulumi packages. " +
			"This tool relies on a Pulumi package's schema spec. " +
			"This tool will not generate the schema.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logging.InitLogging(logToStderr, verbose, false)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			glog.Flush()
		},
	}

	rootCmd.PersistentFlags().BoolVar(&logToStderr, "logtostderr", false, "Log to stderr instead of to files")
	rootCmd.PersistentFlags().IntVarP(&verbose, "verbose", "v", 0, "Enable verbose logging (e.g., v=3); anything >3 is very verbose")

	rootCmd.AddCommand(docs.ResourceDocsCmd())
	rootCmd.AddCommand(metadata.PackageMetadataCmd())

	return rootCmd
}
