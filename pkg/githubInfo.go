package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/pkg/errors"
)

func GetGitHubAPI(path string) (*http.Response, error) {
	token := os.Getenv("GITHUB_TOKEN")
	url := fmt.Sprintf("https://api.github.com%s", path)

	client, err := github_ratelimit.NewRateLimitWaiterClient(nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating GitHub rate limiter client")
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}

	req.Header = http.Header{
		"Content-Type": {"application/json"},
	}

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	return client.Do(req)
}

type GitHubTag struct {
	Name       string `json:"name"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
	Commit     struct {
		Sha string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	NodeID string `json:"node_id"`
}

type GitHubCommit struct {
	Sha    string `json:"sha"`
	NodeID string `json:"node_id"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type RepositoryContent struct {
	Type *string `json:"type,omitempty"`
	// Target is only set if the type is "symlink" and the target is not a normal file.
	// If Target is set, Path will be the symlink path.
	Target   *string `json:"target,omitempty"`
	Encoding *string `json:"encoding,omitempty"`
	Size     *int    `json:"size,omitempty"`
	Name     *string `json:"name,omitempty"`
	Path     *string `json:"path,omitempty"`
	// Content contains the actual file content, which may be encoded.
	// Callers should call GetContent which will decode the content if
	// necessary.
	Content         *string `json:"content,omitempty"`
	SHA             *string `json:"sha,omitempty"`
	URL             *string `json:"url,omitempty"`
	GitURL          *string `json:"git_url,omitempty"`
	HTMLURL         *string `json:"html_url,omitempty"`
	DownloadURL     *string `json:"download_url,omitempty"`
	SubmoduleGitURL *string `json:"submodule_git_url,omitempty"`
}

func GetGitHubFileContents(repoSlug, repoPath, version string) (result *[]RepositoryContent, err error) {
	path := fmt.Sprintf("/repos/%s/contents/%s", repoSlug, repoPath)
	if version != "" {
		path += fmt.Sprintf("?ref=%s", version)
	}
	return getGitHubFileContents(path)
}

func getGitHubFileContents(path string) (result *[]RepositoryContent, err error) {
	rawJSON, err := GetGitHubAPI(strings.TrimPrefix(path, "https://api.github.com"))
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("getting content for path %s", path))
		return
	}
	defer rawJSON.Body.Close()
	if rawJSON.StatusCode != 200 {
		err = fmt.Errorf("getting content for path %s failed with HTTP %d", path, rawJSON.StatusCode)
		return
	}

	var content []RepositoryContent
	err = json.NewDecoder(rawJSON.Body).Decode(&content)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting content for path %s", path))
	}

	fileResult := []RepositoryContent{}
	for _, c := range content {
		if strings.EqualFold(*c.Type, "dir") {
			r, err := getGitHubFileContents(*c.URL)
			if err != nil {
				return nil, err
			}
			fileResult = append(fileResult, (*r)...)
		} else {
			fileResult = append(fileResult, c)
		}
	}

	result = &fileResult
	return
}

func GetGitHubTags(repoSlug string) ([]GitHubTag, error) {
	path := fmt.Sprintf("/repos/%s/tags", repoSlug)
	tagsResp, err := GetGitHubAPI(path)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("getting tags info for %s", repoSlug))
	}
	defer tagsResp.Body.Close()

	if tagsResp.StatusCode != 200 {
		respBody, err := io.ReadAll(tagsResp.Body)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("getting tags info for %s: %s", repoSlug, tagsResp.Status))
		}

		return nil, fmt.Errorf("getting tags info for %s: %s", repoSlug, string(respBody))
	}

	var tags []GitHubTag
	err = json.NewDecoder(tagsResp.Body).Decode(&tags)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("constructing tags information for %s", repoSlug))
	}

	return tags, nil
}
