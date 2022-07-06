package pkg

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	docsgen "github.com/pulumi/pulumi/pkg/v3/codegen/docs"
	"github.com/pulumi/pulumi/pkg/v3/codegen/dotnet"
	go_gen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	"github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pschema "github.com/pulumi/pulumi/pkg/v3/codegen/schema"
)

const (
	tool                        = "Pulumi Docs Generator"
	registryRepo                = "https://github.com/pulumi/registry"
	defaultSchemaFilePathFormat = "/provider/cmd/pulumi-resource-%s/schema.json"
)

var (
	// mainSpec represents a package's original schema. It's called "main" because a package
	// could have a hand-authored overlays schema spec in the overlays folder that could be
	// merged into it.
	mainSpec *pschema.PackageSpec
	//go:embed overlays/**/*.json
	overlays embed.FS
)

func getRepoSlug(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parsing repo url %s: %w", repoURL, err)
	}

	return u.Path, nil
}

func getPulumiPackageFromSchema(docsOutDir string) (*pschema.Package, error) {
	overlaysSchemaFile, err := getOverlaySchema()
	if err != nil {
		return nil, fmt.Errorf("getting overlays schema: %w", err)
	}

	if overlaysSchemaFile != nil {
		overlaySpec := &pschema.PackageSpec{}

		if err := json.Unmarshal(overlaysSchemaFile, overlaySpec); err != nil {
			return nil, fmt.Errorf("unmarshalling overlay schema into a PackageSpec: %w", err)
		}

		if err := mergeOverlaySchemaSpec(mainSpec, overlaySpec); err != nil {
			return nil, fmt.Errorf("merging the overlay schema spec with the main spec: %w", err)
		}
	}

	// Delete existing docs before generating new ones.
	if err := os.RemoveAll(docsOutDir); err != nil {
		return nil, fmt.Errorf("deleting provider directory %v: %w", docsOutDir, err)
	}

	pulPkg, err := pschema.ImportSpec(*mainSpec, nil)
	if err != nil {
		return nil, fmt.Errorf("error importing package spec: %w", err)
	}

	docsgen.Initialize(tool, pulPkg)

	return pulPkg, nil
}

// getOverlaySchema returns the overlay file contents for the package.
// Returns nil if there is no overlay file for the package.
func getOverlaySchema() ([]byte, error) {
	// Test the expected path for an overlays file. If there is no such file, assume
	// that the package has no overlays.
	overlayFilePath := filepath.Join("overlays", mainSpec.Name, "overlays.json")
	f, err := overlays.Open(overlayFilePath)
	if err != nil {
		pathErr := err.(*fs.PathError)
		if pathErr.Err == fs.ErrNotExist {
			overlayFilePath = ""
		} else {
			return nil, fmt.Errorf("checking embedded overlays fs for overlay file: %w", err)
		}
	}

	if overlayFilePath == "" {
		return nil, nil
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading overlay file from embedded fs: %w", err)
	}

	return b, nil
}

func GenerateDocs(host, repoURL, version, schemaFile, docsOutDir, packageTreeJSONOutDir string) error {
	repoSlug, err := getRepoSlug(repoURL)
	if err != nil {
		return err
	}

	// we should be able to take the repo URL + the version + the schema url and
	// construct a file that we can download and read
	schemaFilePath := fmt.Sprintf("%s/%s/%s/%s", host, repoSlug, version, schemaFile)
	resp, err := http.Get(schemaFilePath)
	if err != nil {
		return fmt.Errorf("downloading schema file from %s: %w", schemaFile, err)
	}

	defer resp.Body.Close()
	schema, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading contents of schema file: %w", err)
	}

	// The source schema can be in YAML format. If that's the case
	// convert it to JSON first.
	if strings.HasSuffix(schemaFile, ".yaml") {
		schema, err = yaml.YAMLToJSON(schema)
		if err != nil {
			return fmt.Errorf("reading YAML schema: %w", err)
		}
	}

	mainSpec = &pschema.PackageSpec{}
	if err := json.Unmarshal(schema, mainSpec); err != nil {
		return fmt.Errorf("unmarshalling schema into a PackageSpec: %w", err)
	}
	mainSpec.Version = version

	pulPkg, err := getPulumiPackageFromSchema(docsOutDir)
	if err != nil {
		return fmt.Errorf("generating package from schema file: %w", err)
	}

	if err := generateDocsFromSchema(docsOutDir, pulPkg); err != nil {
		return fmt.Errorf("generating docs from schema: %w", err)
	}

	if err := generatePackageTree(packageTreeJSONOutDir, pulPkg.Name); err != nil {
		return fmt.Errorf("generating package tree: %w", err)
	}

	return nil
}

// mergeOverlaySchemaSpec merges the resources, types and language info from the overlay schema spec
// into the main package spec.
func mergeOverlaySchemaSpec(mainSpec *pschema.PackageSpec, overlaySpec *pschema.PackageSpec) error {
	// Merge the overlay schema spec into the main schema spec.
	for key, value := range overlaySpec.Types {
		if _, ok := mainSpec.Types[key]; ok {
			continue
		}
		mainSpec.Types[key] = value
	}
	for key, value := range overlaySpec.Resources {
		if _, ok := mainSpec.Resources[key]; ok {
			continue
		}
		mainSpec.Resources[key] = value
	}
	for lang, overlayLanguageInfo := range overlaySpec.Language {
		switch lang {
		case "go":
			var mainSchemaPkgInfo go_gen.GoPackageInfo
			if err := json.Unmarshal(mainSpec.Language[lang], &mainSchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling Go package info from the main schema spec: %w", err)
			}

			var overlaySchemaPkgInfo go_gen.GoPackageInfo
			if err := json.Unmarshal(overlayLanguageInfo, &overlaySchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling Go package info from the overlay schema spec: %w", err)
			}

			for key, value := range overlaySchemaPkgInfo.ModuleToPackage {
				if _, ok := mainSchemaPkgInfo.ModuleToPackage[key]; ok {
					continue
				}
				mainSchemaPkgInfo.ModuleToPackage[key] = value
			}

			// Override the language info for Go in the main schema spec.
			b, err := json.Marshal(mainSchemaPkgInfo)
			if err != nil {
				return fmt.Errorf("error marshalling Go package info: %w", err)
			}
			mainSpec.Language[lang] = b
		case "nodejs":
			var mainSchemaPkgInfo nodejs.NodePackageInfo
			if err := json.Unmarshal(mainSpec.Language[lang], &mainSchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling NodeJS package info from the main schema spec: %w", err)
			}

			var overlaySchemaPkgInfo nodejs.NodePackageInfo
			if err := json.Unmarshal(overlayLanguageInfo, &overlaySchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling NodeJS package info from the overlay schema spec: %w", err)
			}

			for key, value := range overlaySchemaPkgInfo.ModuleToPackage {
				if _, ok := mainSchemaPkgInfo.ModuleToPackage[key]; ok {
					continue
				}
				mainSchemaPkgInfo.ModuleToPackage[key] = value
			}

			// Override the language info for NodeJS in the main schema spec.
			b, err := json.Marshal(mainSchemaPkgInfo)
			if err != nil {
				return fmt.Errorf("error marshalling NodeJS package info: %w", err)
			}
			mainSpec.Language[lang] = b
		case "csharp":
			var mainSchemaPkgInfo dotnet.CSharpPackageInfo
			if err := json.Unmarshal(mainSpec.Language[lang], &mainSchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling C# package info from the main schema spec: %w", err)
			}

			var overlaySchemaPkgInfo dotnet.CSharpPackageInfo
			if err := json.Unmarshal(overlayLanguageInfo, &overlaySchemaPkgInfo); err != nil {
				return fmt.Errorf("error un-marshalling C# package info from overlay schema spec: %w", err)
			}

			for key, value := range overlaySchemaPkgInfo.Namespaces {
				if _, ok := mainSchemaPkgInfo.Namespaces[key]; ok {
					continue
				}
				mainSchemaPkgInfo.Namespaces[key] = value
			}
			// Override the language info for C# in the main schema spec.
			b, err := json.Marshal(mainSchemaPkgInfo)
			if err != nil {
				return fmt.Errorf("error marshalling C# package info: %w", err)
			}
			mainSpec.Language[lang] = b
		}
	}

	return nil
}

func generateDocsFromSchema(outDir string, pulPkg *pschema.Package) error {
	files, err := docsgen.GeneratePackage(tool, pulPkg)
	if err != nil {
		return fmt.Errorf("generating Pulumi package: %w", err)
	}

	for f, contents := range files {
		if err := EmitFile(outDir, f, contents); err != nil {
			return fmt.Errorf("emitting file %v: %w", f, err)
		}
	}
	return nil
}

func generatePackageTree(outDir string, pkgName string) error {
	tree, err := docsgen.GeneratePackageTree()
	if err != nil {
		return fmt.Errorf("generating the package tree: %w", err)
	}

	b, err := json.Marshal(tree)
	if err != nil {
		return fmt.Errorf("marshalling the package tree: %w", err)
	}

	filename := fmt.Sprintf("%s.json", pkgName)
	if err := EmitFile(outDir, filename, b); err != nil {
		return fmt.Errorf("writing the package tree: %w", err)
	}

	return nil
}
