package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const defaultReleasesAPI = "https://api.github.com/repos/nabilrn/MyPaas/releases?per_page=20"

var fullGitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type ReleaseStatus struct {
	State           string `json:"state"`
	Channel         string `json:"channel"`
	CurrentBuildSHA string `json:"current_build_sha,omitempty"`
	CurrentTag      string `json:"current_tag,omitempty"`
	LatestTag       string `json:"latest_tag,omitempty"`
	LatestSHA       string `json:"latest_sha,omitempty"`
	ReleaseURL      string `json:"release_url,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
	TrackingRef     string `json:"tracking_ref,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
}

type githubRelease struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	HTMLURL         string `json:"html_url"`
	Draft           bool   `json:"draft"`
	PublishedAt     string `json:"published_at"`
}

func configuredUpdateChannel() string {
	channel := strings.ToLower(strings.TrimSpace(os.Getenv("MYPAAS_UPDATE_CHANNEL")))
	if channel == "" {
		return "release"
	}
	return channel
}

func configuredTrackingRef() string {
	ref := strings.TrimSpace(os.Getenv("MYPAAS_REF"))
	if ref == "" {
		return "main"
	}
	return ref
}

func releaseStatus(ctx context.Context, currentBuildSHA string) (ReleaseStatus, error) {
	currentBuildSHA = strings.TrimSpace(currentBuildSHA)
	channel := configuredUpdateChannel()
	status := ReleaseStatus{
		State:           "unknown",
		Channel:         channel,
		CurrentBuildSHA: currentBuildSHA,
	}

	switch channel {
	case "ref":
		status.State = "tracking_ref"
		status.TrackingRef = configuredTrackingRef()
		return status, nil
	case "release":
		// Continue below.
	default:
		status.State = "unavailable"
		return status, fmt.Errorf("unsupported update channel %q", channel)
	}

	releases, err := fetchPublishedReleases(ctx)
	if err != nil {
		status.State = "unavailable"
		return status, err
	}
	if len(releases) == 0 {
		status.State = "unavailable"
		return status, errors.New("no published MyPaas releases found")
	}

	latest := releases[0]
	latestSHA := strings.TrimSpace(latest.TargetCommitish)
	if !fullGitSHA.MatchString(latestSHA) {
		status.State = "unavailable"
		return status, fmt.Errorf("latest release %s is not pinned to an immutable commit SHA", latest.TagName)
	}

	status.LatestTag = latest.TagName
	status.LatestSHA = strings.ToLower(latestSHA)
	status.ReleaseURL = latest.HTMLURL
	status.PublishedAt = latest.PublishedAt

	for _, release := range releases {
		if currentBuildSHA != "" && strings.EqualFold(strings.TrimSpace(release.TargetCommitish), currentBuildSHA) {
			status.CurrentTag = release.TagName
			break
		}
	}

	if currentBuildSHA == "" || !fullGitSHA.MatchString(currentBuildSHA) {
		status.State = "unknown"
		return status, nil
	}
	if strings.EqualFold(currentBuildSHA, latestSHA) {
		status.State = "current"
		return status, nil
	}

	status.State = "available"
	status.UpdateAvailable = true
	return status, nil
}

func fetchPublishedReleases(ctx context.Context) ([]githubRelease, error) {
	endpoint := strings.TrimSpace(os.Getenv("MYPAAS_RELEASES_API_URL"))
	if endpoint == "" {
		endpoint = defaultReleasesAPI
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build releases request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MyPaas-release-checker")

	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch releases: unexpected HTTP %d", res.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases: %w", err)
	}

	published := releases[:0]
	for _, release := range releases {
		if release.Draft || strings.TrimSpace(release.TagName) == "" {
			continue
		}
		published = append(published, release)
	}
	sort.SliceStable(published, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339, published[i].PublishedAt)
		right, rightErr := time.Parse(time.RFC3339, published[j].PublishedAt)
		if leftErr != nil || rightErr != nil {
			return i < j
		}
		return left.After(right)
	})
	return published, nil
}
