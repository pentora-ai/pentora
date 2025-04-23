package parser

type ServiceInfo struct {
	Name    string
	Version string
}

type Plugin interface {
	Match(banner string) bool
	Extract(banner string) *ServiceInfo
}

var registry []Plugin

func Register(p Plugin) {
	registry = append(registry, p)
}

func Dispatch(banner string) *ServiceInfo {
	for _, plugin := range registry {
		if plugin.Match(banner) {
			return plugin.Extract(banner)
		}
	}
	return nil
}
