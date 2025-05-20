// cmd/configuration.go
package cmd

import (
	"github.com/pentora-ai/pentora/pkg/config/static"
	"github.com/pentora-ai/pentora/pkg/types"
)

// PentoraCmdConfiguration wrapss the static configuration and extra parameters
type PentoraCmdConfiguration struct {
	static.Configuration `export:"true"`
}

// NewPentoraConfiguration creates a PentoraCmdConfiguration with default values.
func NewPentoraConfiguration() *PentoraCmdConfiguration {
	return &PentoraCmdConfiguration{
		Configuration: static.Configuration{
			Global: &static.Global{
				CheckNewVersion: true,
			},
			Log: &types.PentoraLog{
				Level: "debug",
			},
		},
	}
}
