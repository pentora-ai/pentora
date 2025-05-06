package version

import "time"

var (
	// Version holds the current version of pentora.
	Version = "dev"
	// Codename holds the current version codename of pentora.
	Codename = "cheddar" // beta cheese
	// BuildDate holds the build date of pentora.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of pentora.
	StartDate = time.Now()
	// DisableDashboardAd disables ad in the dashboard.
	DisableDashboardAd = false
)
