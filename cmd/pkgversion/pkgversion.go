package pkgversion

import (
	"fmt"
	"github.com/spf13/cobra"
)

func CheckVersion() *cobra.Command {

	var repoSlug string
	var schemaFile string
	cmd := &cobra.Command{
		Use:   "pkgversion",
		Short: "Check a Pulumi package version",
		Long:  `Get the most recent version of a Pulumi package on the registry`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Checking the correct version on github")
			fmt.Println(repoSlug)
			fmt.Println(schemaFile)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repoSlug, "repoSlug", "r", "", "The repository slug e.g. pulumi/pulumi-provider") //TODO: consider the name and if we need full link here
	cmd.Flags().StringVarP(&schemaFile, "schemaFile", "s", "", "Relative path to the schema.json file from "+
		"the root of the repository.")
	cmd.MarkFlagRequired("schemaFile")
	cmd.MarkFlagRequired("repoSlug")
	return cmd
}
