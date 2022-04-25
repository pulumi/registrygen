package pkgversion

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/pulumi/registrygen/pkg"
	"github.com/spf13/cobra"

	"io/ioutil"
	"net/http"
	"strings"
)

func CheckVersion() *cobra.Command {

	var repoSlug string
	cmd := &cobra.Command{
		Use:   "pkgversion",
		Short: "Check a Pulumi package version",
		Long:  `Get the most recent version of a Pulumi package and compare with the version in the registry`,
		RunE: func(cmd *cobra.Command, args []string) error {
			latest := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoSlug)
			version, err := getLatestVersion(latest)
			if err != nil {
				return err
			}

			repoName := ""
			githubSlugParts := strings.Split(repoSlug, "/")
			if len(githubSlugParts) > 0 {
				repoName = githubSlugParts[1]
			}

			pkgName := strings.TrimPrefix(repoName, "pulumi-")
			pkgMetadata := fmt.Sprintf("https://raw.githubusercontent.com/pulumi/registry/master/themes/default/data/registry/packages/%s.yaml", pkgName)
			regVersion, err := getRegistryVersion(pkgMetadata)
			if err != nil {
				return err
			}

			// print version tag if there's a difference, and not if there isn't
			// we assume that the published latest version from the provider repo is the desired one, so any difference
			// between versions should indicate an update to the registry version
			if version != regVersion {
				fmt.Println(version)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&repoSlug, "repoSlug", "", "The repository slug e.g. pulumi/pulumi-provider")
	cmd.MarkFlagRequired("repoSlug")
	return cmd
}

func getLatestVersion(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("getting latest version from %s", url))
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.Wrap(err, fmt.Sprintf("Could not find a release at %s", url))
	}

	var tag pkg.GitHubTag
	err = json.NewDecoder(resp.Body).Decode(&tag)

	if err != nil {
		return "", errors.Wrap(err, "failure reading contents of latest tag")
	}

	return tag.Name, nil
}

func getRegistryVersion(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("getting latest version from %s", url))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.Wrap(err, "file not found")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failure reading contents of remote file")
	}

	var meta pkg.PackageMeta
	err = yaml.Unmarshal(contents, &meta)
	if err != nil {
		return "", errors.Wrap(err, "error unmarshalling yaml file")
	}

	return meta.Version, nil
}
