package command

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type HTTPClient interface {
	Do(r *http.Request) (*http.Response, error)
}

type GithubReleaseDownloader struct {
	log Logger
	c   HTTPClient
}

func NewGithubReleaseDownloader(c HTTPClient, log Logger) GithubReleaseDownloader {
	return GithubReleaseDownloader{
		log: log,
		c:   c,
	}
}

func (d GithubReleaseDownloader) Download(assetName string) string {
	releases := d.getReleases()

	sort.Sort(githubReleases(releases))
	for _, release := range releases {
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				tmp, err := ioutil.TempDir("", asset.Name)
				if err != nil {
					d.log.Fatalf("failed to create temp directory: %s", err)
				}
				p := path.Join(tmp, asset.Name)
				d.downloadAsset(asset.Name, asset.BrowserDownloadURL, p)
				return p
			}
		}
	}

	d.log.Fatalf("unable to find %s asset in releases", assetName)
	return ""
}

func (d GithubReleaseDownloader) getReleases() githubReleases {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/cloudfoundry/cf-drain-cli/releases", nil)
	if err != nil {
		d.log.Fatalf("failed to create request to github: %s", err)
	}

	resp, err := d.c.Do(req)
	if err != nil {
		d.log.Fatalf("failed to read from github: %s", err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		d.log.Fatalf("unexpected status code (%d) from github", resp.StatusCode)
	}

	var releases githubReleases
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		d.log.Fatalf("failed to decode releases response from github")
	}

	return releases
}

func (d GithubReleaseDownloader) downloadAsset(assetName, URL, p string) {
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		d.log.Fatalf("failed to create request to github: %s", err)
	}

	resp, err := d.c.Do(req)
	if err != nil {
		d.log.Fatalf("failed to read from github: %s", err)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		d.log.Fatalf("unexpected status code (%d) from github", resp.StatusCode)
	}

	f, err := os.Create(p)
	if err != nil {
		d.log.Fatalf("failed to create temp file: %s", err)
	}
	defer func() {
		f.Close()
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		d.log.Fatalf("failed to read github asset: %s", err)
	}

	err = f.Chmod(os.ModePerm)
	if err != nil {
		d.log.Fatalf("failed to make %s executable: %s", assetName, err)
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type githubReleases []githubRelease

func (r githubReleases) Len() int {
	return len(r)
}

func (r githubReleases) Swap(i, j int) {
	t := r[i]
	r[i] = r[j]
	r[j] = t
}

func (r githubReleases) Less(a, b int) bool {
	ta := r.convertToInts(r[a].TagName)
	tb := r.convertToInts(r[b].TagName)

	for i := range tb {
		if i > len(ta) {
			return false
		}

		if tb[i] == ta[i] {
			continue
		}

		return tb[i] < ta[i]
	}

	return ta[len(tb)-1] != 0
}

func (r githubReleases) convertToInts(tagName string) []uint64 {
	s := strings.Split(strings.Trim(tagName, "v"), ".")

	var result []uint64
	for _, ss := range s {
		// Non-numeric values get to be a 0
		u, _ := strconv.ParseUint(ss, 10, 64)
		result = append(result, u)
	}
	return result
}
