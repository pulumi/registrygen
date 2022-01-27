package cmd

import (
	"github.com/pulumi/registrygen/cmd/docs"
	"github.com/pulumi/registrygen/cmd/metadata"
	"github.com/pulumi/registrygen/cmd/version"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "registrygen",
		Short: "Generate Package Metadata and API Docs for the Pulumi registry",
		Long: "A tool to generate API docs and package metadata for Pulumi packages. " +
			"This tool relies on a Pulumi package's schema spec. " +
			"This tool will not generate the schema.",
	}

	rootCmd.AddCommand(metadata.PackageMetadataCmd())
	rootCmd.AddCommand(version.Command())
	rootCmd.AddCommand(docs.GenerateCommand())

	return rootCmd
}
