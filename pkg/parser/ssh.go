package parser

import "strings"

type sshPlugin struct{}

func (p *sshPlugin) Match(banner string) bool {
	return strings.HasPrefix(banner, "SSH-")
}

func (p *sshPlugin) Extract(banner string) *ServiceInfo {
	// Check if the banner starts with "SSH-"
	if !p.Match(banner) {
		return nil
	}

	// Parse the SSH version from the banner
	version := extractSSHVersion(banner)
	if version == "" {
		return nil
	}

	return &ServiceInfo{
		Name:    "SSH",
		Version: version,
	}
}

// sshParser is a simple parser for SSH banners.
// It checks if the banner starts with "SSH-" and extracts the version.

// extractSSHVersion parses banner like "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.3"
// and returns "OpenSSH_8.9p1" as version string
func extractSSHVersion(banner string) string {
	parts := strings.Split(banner, "-")
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func init() {
	Register(&sshPlugin{})
}
