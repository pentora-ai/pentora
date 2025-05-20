// pkg/version/version.go
// Package version provides version metadata for the application.
package version

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v28/github"
	goversion "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
)

// These variables are typically injected at build time using -ldflags
var (
	// Version holds the current version of pentora.
	Version = "dev"
	// Commit holds the current version commit of pentora.
	Commit = "none" // beta cheese
	// BuildDate holds the build date of pentora.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of pentora.
	StartDate = time.Now()
)

// Struct returns version information in a structured format.
type Struct struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// Info returns a formatted version string.
func Info() string {
	return fmt.Sprintf("Pentora %s (commit: %s, date: %s)", Version, Commit, BuildDate)
}

// Get returns version information as a Struct.
func Get() Struct {
	return Struct{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

// CheckNewVersion checks if a new version is available.
func CheckNewVersion() {
	if Version == "dev" {
		return
	}

	client := github.NewClient(nil)

	updateURL, err := url.Parse("https://update.pentora.ai/")
	if err != nil {
		log.Warn().Err(err).Msg("Error checking new version")
		return
	}
	client.BaseURL = updateURL

	releases, resp, err := client.Repositories.ListReleases(context.Background(), "pentora-ai", "pentora", nil)
	if err != nil {
		log.Warn().Err(err).Msg("Error checking new version")
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().Msgf("Error checking new version: status=%s", resp.Status)
		return
	}

	currentVersion, err := goversion.NewVersion(Version)
	if err != nil {
		log.Warn().Err(err).Msg("Error checking new version")
		return
	}

	for _, release := range releases {
		releaseVersion, err := goversion.NewVersion(*release.TagName)
		log.Warn().Msg(releaseVersion.String())
		if err != nil {
			log.Warn().Err(err).Msg("Error checking new version")
			return
		}

		if len(currentVersion.Prerelease()) == 0 && len(releaseVersion.Prerelease()) > 0 {
			continue
		}

		if releaseVersion.GreaterThan(currentVersion) {
			log.Warn().Err(err).Msgf("A new release of Pentora has been found: %s. Please consider updating.", releaseVersion.String())
			return
		}
	}
}
