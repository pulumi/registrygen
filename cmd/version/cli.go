package version

import (
	"fmt"

	cliVersion "github.com/pulumi/registrygen/pkg/version"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "version",
		Short: "Get the current version",
		Long:  `Get the current version of pulumictl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(cliVersion.Version)
			return nil
		},
	}
	return command
}
