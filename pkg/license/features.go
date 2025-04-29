package license

// Default set of features allowed in "free" mode (no license)
var defaultFreeFeatures = []string{
	"scanner",
	"parser",
	"logview",
	"search",
	"alerts",
	"export",
	"help",
	"json-output",
	"html-report",
	"syslog-validate",
	"raw-analyze",
}

// Internally mapped for fast access
var defaultFreeMap = make(map[string]bool)

func init() {
	for _, f := range defaultFreeFeatures {
		defaultFreeMap[f] = true
	}
}

// IsFeatureAllowed returns true if the feature is allowed under current license status
func IsFeatureAllowed(status *LicenseStatus, feature string) bool {
	if status != nil && status.Valid {
		return status.Payload.HasFeature(feature)
	}
	return defaultFreeMap[feature]
}

func GetFreeFeatures() []string {
	return defaultFreeFeatures
}
