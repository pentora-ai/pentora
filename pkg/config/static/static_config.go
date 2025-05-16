// pkg/config/static/static_config.go
package static

import "github.com/pentora-ai/pentora/pkg/types"

const (
// DefaultConfigPath is the default path for the configuration file
)

// Configuration is the static configuration.
type Configuration struct {
	Global *Global `description:"Global configuration options" json:"global,omitempty" yaml:"global,omitempty" export:"true"`

	API *API              `description:"Enable api/dashboard." json:"api,omitempty" yaml:"api,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	Log *types.PentoraLog `description:"Traefik log settings." json:"log,omitempty" yaml:"log,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// Global holds the global configuration.
type Global struct {
	CheckNewVersion    bool `description:"Periodically check if a new version has been released." json:"checkNewVersion,omitempty" yaml:"checkNewVersion,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	SendAnonymousUsage bool `description:"Periodically send anonymous usage statistics. If the option is not specified, it will be disabled by default." json:"sendAnonymousUsage,omitempty" yaml:"sendAnonymousUsage,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// API holds the API configuration.
type API struct {
	BasePath  string `description:"Defines the base path where the API and Dashboard will be exposed." json:"basePath,omitempty" yaml:"basePath,omitempty" export:"true"`
	Dashboard bool   `description:"Activate dashboard." json:"dashboard,omitempty" yaml:"dashboard,omitempty" export:"true"`
	Debug     bool   `description:"Enable additional endpoints for debugging and profiling." json:"debug,omitempty" yaml:"debug,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (a *API) SetDefaults() {
	a.BasePath = "/"
	a.Dashboard = true
}
