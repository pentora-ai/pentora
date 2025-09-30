package fingerprint

import "sync"

var (
	fingerprinterMu  sync.RWMutex
	fingerprinterSet []Fingerprinter
)

func init() {
	fingerprinterSet = make([]Fingerprinter, 0)
}

// RegisterFingerprinter adds a fingerprinter to the global catalog.
func RegisterFingerprinter(fp Fingerprinter) {
	fingerprinterMu.Lock()
	defer fingerprinterMu.Unlock()
	fingerprinterSet = append(fingerprinterSet, fp)
}

// ListFingerprinters returns a snapshot of the registered fingerprinters.
func ListFingerprinters() []Fingerprinter {
	fingerprinterMu.RLock()
	defer fingerprinterMu.RUnlock()
	return append([]Fingerprinter(nil), fingerprinterSet...)
}

// NewDefaultCoordinator returns a Coordinator pre-populated with all registered fingerprinters.
func NewDefaultCoordinator() *Coordinator {
	fingerprinterMu.RLock()
	defer fingerprinterMu.RUnlock()
	return NewCoordinator(fingerprinterSet...)
}
