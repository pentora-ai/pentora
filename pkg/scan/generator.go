// pkg/scan/generator.go
package scan

import (
	"github.com/google/uuid"
)

// GenerateScanID returns a new UUID-based scan ID.
func GenerateScanID() string {
	return uuid.NewString()
}
