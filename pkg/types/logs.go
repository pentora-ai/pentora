// pkg/types/logs.go
package types

// PentoraLog holds the configuration settings for the pentora logger.
type PentoraLog struct {
	Level   string `description:"Log level set to pentora logs." json:"level,omitempty" yaml:"level,omitempty" export:"true"`
	Format  string `description:"Pentora log format: json | common" json:"format,omitempty" yaml:"format,omitempty" export:"true"`
	NoColor bool   `description:"When using the 'common' format, disables the colorized output." json:"noColor,omitempty" yaml:"noColor,omitempty" export:"true"`
}
