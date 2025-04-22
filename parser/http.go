package parser

import (
	"regexp"
	"strings"
)

type httpPlugin struct{}

func (p *httpPlugin) Match(banner string) bool {
	return strings.Contains(strings.ToLower(banner), "server:")
}

func (p *httpPlugin) Extract(banner string) *ServiceInfo {
	lines := strings.Split(banner, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "server:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info := strings.TrimSpace(parts[1])
				re := regexp.MustCompile(`(?i)([a-zA-Z0-9._\-]+)[/ ]?([0-9.a-zA-Z_\-]*)`)
				matches := re.FindStringSubmatch(info)
				if len(matches) >= 2 {
					name := matches[1]
					version := ""
					if len(matches) > 2 {
						version = matches[2]
					}
					return &ServiceInfo{Name: name, Version: version}
				}
			}
		}
	}
	return nil
}

func init() {
	Register(&httpPlugin{})
}
