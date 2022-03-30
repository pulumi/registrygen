package sdk

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	dotnetgen "github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
	gogen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	nodejsgen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pythongen "github.com/pulumi/pulumi/pkg/v3/codegen/python"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tools"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

func GenerateSdkCommand() *cobra.Command {
	var schemaPath string
	var providerName string
	var language string

	cmd := &cobra.Command{
		Use:   "sdk <language>",
		Short: "Generate SDK for an Node.js",
		RunE: func(cmd *cobra.Command, args []string) error {

			if schemaPath == "" {
				schemaPath = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/provider/cmd/pulumi-resource-%[2]s/schema.json",
					"pulumi", providerName)
			}

			pkgSpec, err := downloadSchema(schemaPath)
			if err != nil {
				return err
			}

			return emitPackage(pkgSpec, language, "./sdk/"+language)
		},
	}

	cmd.Flags().StringVarP(&schemaPath, "schemaPath", "s", "", "Relative path to the schema.json file from "+
		"the root of the repository. If not schemaFile is specified, then providerName is required so the schemaFile path can "+
		"be inferred to be provider/cmd/pulumi-resource-<providerName>/schema.json")
	cmd.Flags().StringVar(&providerName, "providerName", "", "The name of the provider e.g. aws, aws-native. "+
		"Required when there is no schemaFile flag specified.")
	cmd.Flags().StringVar(&language, "language", "", "")

	return cmd
}

func generate(ppkg *schema.Package, language string) (map[string][]byte, error) {
	toolDescription := "the Pulumi SDK Generator"
	extraFiles := map[string][]byte{}
	switch language {
	case "nodejs":
		return nodejsgen.GeneratePackage(toolDescription, ppkg, extraFiles)
	case "python":
		return pythongen.GeneratePackage(toolDescription, ppkg, extraFiles)
	case "go":
		return gogen.GeneratePackage(toolDescription, ppkg)
	case "dotnet":
		return dotnetgen.GeneratePackage(toolDescription, ppkg, extraFiles)
	}

	return nil, errors.Errorf("unknown language '%s'", language)
}

func emitPackage(pkgSpec *schema.PackageSpec, language, outDir string) error {
	ppkg, err := schema.ImportSpec(*pkgSpec, nil)
	if err != nil {
		return errors.Wrap(err, "reading schema")
	}

	files, err := generate(ppkg, language)
	if err != nil {
		return errors.Wrapf(err, "generating %s package", language)
	}

	for f, contents := range files {
		if err := emitFile(outDir, f, contents); err != nil {
			return errors.Wrapf(err, "emitting file %v", f)
		}
	}

	return nil
}

// emitFile creates a file in a given directory and writes the byte contents to it.
func emitFile(outDir, relPath string, contents []byte) error {
	p := path.Join(outDir, relPath)
	if err := tools.EnsureDir(path.Dir(p)); err != nil {
		return errors.Wrap(err, "creating directory")
	}

	f, err := os.Create(p)
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	defer contract.IgnoreClose(f)

	_, err = f.Write(contents)
	return err
}

func downloadSchema(schemaUrlOrPath string) (*schema.PackageSpec, error) {
	var body []byte
	if strings.HasPrefix(schemaUrlOrPath, "http") {
		resp, err := http.Get(schemaUrlOrPath)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		b, err := ioutil.ReadFile(schemaUrlOrPath)
		if err != nil {
			return nil, err
		}
		body = b
	}

	var sch schema.PackageSpec
	if err := json.Unmarshal(body, &sch); err != nil {
		return nil, err
	}

	return &sch, nil
}
