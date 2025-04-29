package version

import (
	"io"
	"runtime"
	"text/template"

	"github.com/pentoraai/pentora/pkg/version"
	"github.com/spf13/cobra"
)

var versionTemplate = `Version:      {{.Version}}
Codename:     {{.Codename}}
Go version:   {{.GoVersion}}
Built:        {{.BuildTime}}
OS/Arch:      {{.Os}}/{{.Arch}}`

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Pentora",
	Long:  `All software has versions. This is Pentora's`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return GetPrint(cmd.OutOrStdout())
	},
}

// GetPrint write Printable version.
func GetPrint(wr io.Writer) error {
	tmpl, err := template.New("").Parse(versionTemplate)
	if err != nil {
		return err
	}

	v := struct {
		Version   string
		Codename  string
		GoVersion string
		BuildTime string
		Os        string
		Arch      string
	}{
		Version:   version.Version,
		Codename:  version.Codename,
		GoVersion: runtime.Version(),
		BuildTime: version.BuildDate,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	return tmpl.Execute(wr, v)
}
