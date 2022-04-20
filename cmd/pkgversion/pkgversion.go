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

	var owner string
	var repo string
	cmd := &cobra.Command{
		Use:   "pkgversion",
		Short: "Check a Pulumi package version",
		Long:  `Get the most recent version of a Pulumi package and compare with the version in the registry`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Checking the correct version on github")
			fmt.Println(owner)
			latest := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
			version, err := getLatestVersion(latest)
			if err != nil {
				return err
			}
			pkgName := strings.TrimPrefix(repo, "pulumi-")
			fmt.Println(pkgName)
			pkgMetadata := fmt.Sprintf("https://raw.githubusercontent.com/pulumi/registry/master/themes/default/data/registry/packages/%s.yaml", pkgName)
			regVersion, err := getRegistryVersion(pkgMetadata)
			if err != nil {
				return err
			}
			fmt.Println("Latest version:", version)
			fmt.Println("Registry version:", regVersion)
			// emit version tag if there's a difference, and not if there isn't
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "o", "", "The github owner or organization, e.g. pulumi")

	cmd.Flags().StringVarP(&repo, "repo", "r", "", "The github repo for this package, e.g. pulumi-aws")
	cmd.MarkFlagRequired("owner")
	cmd.MarkFlagRequired("repo")
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
