package docs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/pulumi/registrygen/pkg"
	"github.com/spf13/cobra"
)

func GenerateCommand() *cobra.Command {
	generateCommand := &cobra.Command{
		Use:   "generate",
		Short: "Generate API Docs for the registry",
		Long: "A tool to generate API docs and package metadata for Pulumi packages. " +
			"This tool relies on a Pulumi package's schema spec. " +
			"This tool will not generate the schema.",
	}

	generateCommand.AddCommand(AllPackageDocsCmd())
	generateCommand.AddCommand(PackageDocsCmd())

	return generateCommand
}

func AllPackageDocsCmd() *cobra.Command {
	var registryPackagesPath string
	var baseDocsOutDir string
	var packageTreeJSONOutDir string
	var host string

	cmd := &cobra.Command{
		Use:   "all-docs",
		Short: "Generate API docs for an entire registry",
		RunE: func(cmd *cobra.Command, args []string) error {

			metadataFiles, err := os.ReadDir(registryPackagesPath)
			if err != nil {
				return fmt.Errorf("reading the registry packages dir: %w", err)
			}

			for _, packageMetadata := range metadataFiles {
				metadataFilePath := filepath.Join(registryPackagesPath, packageMetadata.Name())

				b, err := os.ReadFile(metadataFilePath)
				if err != nil {
					return fmt.Errorf("reading the metadata file %s: %w", metadataFilePath, err)
				}

				var metadata pkg.PackageMeta
				if err := yaml.Unmarshal(b, &metadata); err != nil {
					return fmt.Errorf("unmarshalling the metadata file %s: %w", metadataFilePath, err)
				}

				if metadata.RepoURL == "" {
					return fmt.Errorf("metadata for package %q does not contain the repo_url", metadata.Name)
				}

				docsOutDir := filepath.Join(baseDocsOutDir, metadata.Name, "api-docs")
				if err := pkg.GenerateDocs(host, metadata.RepoURL, metadata.Version, metadata.SchemaFilePath, docsOutDir, packageTreeJSONOutDir); err != nil {
					return fmt.Errorf("error generating docs for %s: %w", metadata.Name, err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&registryPackagesPath, "registryPackagesPath", "../registry/themes/default/data/registry/packages/", "The path to the registry metadata files")
	cmd.Flags().StringVar(&baseDocsOutDir, "docsOutDir", "content/registry/packages", "The directory path to where the docs will be written to")
	cmd.Flags().StringVar(&packageTreeJSONOutDir, "packageTreeJSONOutDir", "static/registry/packages/navs", "The directory path to write the "+
		"package tree JSON file to")
	cmd.Flags().StringVar(&host, "host", "https://raw.githubusercontent.com", "The url for source control host")

	return cmd
}

func PackageDocsCmd() *cobra.Command {
	var schemaFile string
	var repoSlug string
	var version string
	var docsOutDir string
	var packageTreeJSONOutDir string
	var host string

	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate API Docs docs from a Pulumi schema file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return pkg.GenerateDocs(host, repoSlug, version, schemaFile, docsOutDir, packageTreeJSONOutDir)
		},
	}

	cmd.Flags().StringVarP(&schemaFile, "schemaFile", "s", "", "Path to the schema.json file")
	cmd.Flags().StringVar(&repoSlug, "repoSlug", "", "The repository slug e.g. pulumi/pulumi-provider")
	cmd.Flags().StringVar(&version, "version", "", "The version of the package")
	cmd.Flags().StringVar(&docsOutDir, "docsOutDir", "", "The directory path to where the docs will be written to")
	cmd.Flags().StringVar(&packageTreeJSONOutDir, "packageTreeJSONOutDir", "", "The directory path to write the "+
		"package tree JSON file to")
	cmd.Flags().StringVar(&host, "host", "https://raw.githubusercontent.com", "The url for source control host")

	cmd.MarkFlagRequired("repoSlug")
	cmd.MarkFlagRequired("docsOutDir")
	cmd.MarkFlagRequired("packageTreeJSONOutDir")
	cmd.MarkFlagRequired("schemaFile")
	cmd.MarkFlagRequired("version")

	return cmd
}
