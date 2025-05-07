// pkg/api/version/version.go

package version

import (
	"encoding/json"
	"net/http"

	"github.com/pentoraai/pentora/pkg/version"
)

var (
	Version = version.Version
	Commit  = version.Codename
	Build   = version.BuildDate
)

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"version": Version,
		"commit":  Commit,
		"build":   Build,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
