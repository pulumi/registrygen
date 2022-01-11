package metadata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	pschema "github.com/pulumi/pulumi/pkg/v3/codegen/schema"
	"github.com/pulumi/registrygen/pkg"
	"github.com/spf13/cobra"
)

const defaultPackageCategory = pkg.PackageCategoryCloud

var mainSpec *pschema.PackageSpec

var featuredPackages = []string{
	"aws",
	"azure-native",
	"gcp",
	"kubernetes",
}

func PackageMetadataCmd() *cobra.Command {
	var repoSlug string
	var categoryStr string
	var component bool
	var publisher string
	var schemaFile string
	var title string
	var version string

	cmd := &cobra.Command{
		Use:   "metadata <args>",
		Short: "Generate package metadata from Pulumi schema",
		RunE: func(cmd *cobra.Command, args []string) error {

			// we should be able to take the repo URL + the version + the schema url and
			// construct a file that we can download and read
			schemaFilePath := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s",
				repoSlug, version, schemaFile)
			resp, err := http.Get(schemaFilePath)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("downloading schema file from %s", schemaFile))
			}

			defer resp.Body.Close()
			schema, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "reading contents of schema file")
			}

			// The source schema can be in YAML format. If that's the case
			// convert it to JSON first.
			if strings.HasSuffix(schemaFile, ".yaml") {
				schema, err = yaml.YAMLToJSON(schema)
				if err != nil {
					return errors.Wrap(err, "reading YAML schema")
				}
			}

			// try and get the version release data using the github releases API
			tagsUrl := fmt.Sprintf("https://api.github.com/repos/%s/tags", repoSlug)

			var tags []pkg.GitHubTag
			tagsResp, err := http.Get(tagsUrl)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("getting tags info for %s", repoSlug))
			}

			defer tagsResp.Body.Close()
			err = json.NewDecoder(tagsResp.Body).Decode(&tags)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("constructing tags information for %s", repoSlug))
			}

			var commitDetails string
			for _, tag := range tags {
				if tag.Name == version {
					commitDetails = tag.Commit.URL
					break
				}
			}

			publishedDate := time.Now()
			if commitDetails != "" {
				var commit pkg.GitHubCommit
				// now let's make a request to the specific commit to get the date
				commitResp, err := http.Get(commitDetails)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("getting release info for %s", repoSlug))
				}

				defer commitResp.Body.Close()
				err = json.NewDecoder(commitResp.Body).Decode(&commit)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("constructing commit information for %s", repoSlug))
				}

				publishedDate = commit.Commit.Author.Date
			}

			mainSpec = &pschema.PackageSpec{}
			if err := json.Unmarshal(schema, mainSpec); err != nil {
				return errors.Wrap(err, "unmarshalling schema into a PackageSpec")
			}
			mainSpec.Version = version

			if mainSpec.Repository == "" {
				return errors.New("repository field must be set in the package schema")
			}

			status := pkg.PackageStatusGA
			if strings.HasPrefix(version, "v0.") {
				status = pkg.PackageStatusPublicPreview
			}

			category, err := getPackageCategory(mainSpec, categoryStr)
			if err != nil {
				return errors.Wrap(err, "getting category")
			}

			// If the title was not overridden, then try to determine
			// the title from the schema.
			if title == "" {
				// If the schema for this package does not have the
				// displayName, then use its package name.
				if mainSpec.DisplayName == "" {
					title = mainSpec.Name
					// Eventually all of Pulumi's own packages will have the displayName
					// set in their schema but for the time being until they are updated
					// with that info, let's lookup the proper title from the lookup map.
					if v, ok := pkg.TitleLookup[mainSpec.Name]; ok {
						title = v
					}
				} else {
					title = mainSpec.DisplayName
				}
			}

			native := mainSpec.Attribution == ""
			// If native is false, check if the schema has the "kind/native" tag in the Keywords
			// array.
			if !native {
				native = isNative(mainSpec.Keywords)
			}

			if !component {
				component = isComponent(mainSpec.Keywords)
			}

			if native && component {
				glog.Warning("Package found to be marked as both native and component. Will proceed with " +
					"tagging the package as a component but not native.")
				native = false
			}

			if publisher == "" && mainSpec.Publisher != "" {
				publisher = mainSpec.Publisher
			} else if publisher == "" {
				publisher = "Pulumi"
			}

			cleanSchemaFilePath := func(s string) string {
				s = strings.ReplaceAll(s, "../", "")
				s = strings.ReplaceAll(s, fmt.Sprintf("pulumi-%s", mainSpec.Name), "")
				return s
			}

			pm := pkg.PackageMeta{
				Name:        mainSpec.Name,
				Description: mainSpec.Description,
				LogoURL:     mainSpec.LogoURL,
				Publisher:   publisher,
				Title:       title,

				RepoURL:        mainSpec.Repository,
				SchemaFilePath: cleanSchemaFilePath(schemaFile),

				PackageStatus: status,
				UpdatedOn:     publishedDate.Unix(),
				Version:       version,

				Category:  category,
				Component: component,
				Featured:  isFeaturedPackage(mainSpec.Name),
				Native:    native,
			}
			b, err := yaml.Marshal(pm)
			if err != nil {
				return errors.Wrap(err, "generating package metadata")
			}

			cwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			metadataFileName := fmt.Sprintf("%s.yaml", mainSpec.Name)
			if err := pkg.EmitFile(filepath.Join(cwd, "output"), metadataFileName, b); err != nil {
				return errors.Wrap(err, "writing metadata file")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&repoSlug, "repoSlug", "", "The repository slug e.g. pulumi/pulumi-provider")
	cmd.Flags().StringVarP(&schemaFile, "schemaFile", "s", "", "Relative path to the schema.json file from "+
		"the root of the repository")
	cmd.Flags().StringVar(&version, "version", "", "The version of the package")
	cmd.Flags().StringVar(&categoryStr, "category", "", fmt.Sprintf("The category for the package. Value must "+
		"match one of the keys in the map: %v", pkg.CategoryNameMap))
	cmd.Flags().StringVar(&publisher, "publisher", "", "The publisher's display name to be shown in the package. "+
		"This will default to Pulumi")
	cmd.Flags().StringVar(&title, "title", "", "The display name of the package. If ommitted, the name of the "+
		"package will be used")
	cmd.Flags().BoolVar(&component, "component", false, "Whether or not this package is a component and not a provider")

	cmd.MarkFlagRequired("schemaFile")
	cmd.MarkFlagRequired("version")
	cmd.MarkFlagRequired("repoSlug")

	return cmd
}

func getPackageCategory(mainSpec *pschema.PackageSpec, categoryOverrideStr string) (pkg.PackageCategory, error) {
	var category pkg.PackageCategory
	var err error

	// If a category override was passed-in, use that instead of what's in the schema.
	if categoryOverrideStr != "" {
		glog.V(2).Infof("Using category override name %s\n", categoryOverrideStr)
		if n, ok := pkg.CategoryNameMap[categoryOverrideStr]; !ok {
			return "", errors.New(fmt.Sprintf("invalid override for category name %s", categoryOverrideStr))
		} else {
			category = n
		}
	} else if c, ok := pkg.CategoryLookup[mainSpec.Name]; ok {
		glog.V(2).Infoln("Using the category for this package from the lookup map")
		// TODO: This condition can be removed when all packages under the `pulumi` org
		// have a proper category tag in their schema.
		category = c
	}

	if category != "" {
		return category, nil
	}

	glog.V(2).Infoln("Looking-up category from the keywords in the schema")
	category, err = getCategoryFromKeywords(mainSpec.Keywords)
	if err != nil {
		return "", errors.Wrap(err, "getting the category from keywords")
	}

	return category, nil
}

// getCategoryFromKeywords searches for a tag in the provided keywords slice
// with a prefix of category/. Returns the converted category type if such a tag
// is found. Otherwise, returns PackageCategoryCloud always as the default.
func getCategoryFromKeywords(keywords []string) (pkg.PackageCategory, error) {
	categoryTag := getTagWithPrefixFromKeywords(keywords, "category/")
	if categoryTag == nil {
		return defaultPackageCategory, nil
	}

	categoryName := strings.Replace(*categoryTag, "category/", "", -1)
	var category pkg.PackageCategory
	if n, ok := pkg.CategoryNameMap[categoryName]; !ok {
		return defaultPackageCategory, errors.New(fmt.Sprintf("invalid category tag %s", *categoryTag))
	} else {
		category = n
	}

	return category, nil
}

func isComponent(keywords []string) bool {
	return getTagFromKeywords(keywords, "kind/component") != nil
}

func isFeaturedPackage(str string) bool {
	for _, v := range featuredPackages {
		if v == str {
			return true
		}
	}
	return false
}

func isNative(keywords []string) bool {
	return getTagFromKeywords(keywords, "kind/native") != nil
}

func getTagWithPrefixFromKeywords(keywords []string, tagPrefix string) *string {
	for _, k := range keywords {
		if strings.HasPrefix(k, tagPrefix) {
			return &k
		}
	}

	glog.V(2).Infof("A tag with the prefix %q was not found in the package's keywords", tagPrefix)
	return nil
}

func getTagFromKeywords(keywords []string, tag string) *string {
	for _, k := range keywords {
		if k == tag {
			return &k
		}
	}

	glog.V(2).Infof("The tag %q was not found in the package's keywords", tag)
	return nil
}