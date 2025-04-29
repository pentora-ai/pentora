package license

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetDefaultLicensePath returns the platform-specific license file path
func GetDefaultLicensePath() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			// fallback
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "Pentora", "license.license")
	}

	return "/etc/pentora/license.license"
}

// GetPublicKeyPath returns the default public key location
func GetPublicKeyPath() string {
	if runtime.GOOS == "windows" {
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "Pentora", "public.pem")
	}

	return "/etc/pentora/public.pem"
}
