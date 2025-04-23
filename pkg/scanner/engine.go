package scanner

import (
	"strings"
	"sync"
	"time"

	"github.com/pentoraai/pentora/pkg/parser"
	"github.com/pentoraai/pentora/pkg/plugin"
)

// ScanJob defines the structure of a scanning task
// It contains the target IP addresses and the ports to scan.
type ScanJob struct {
	GroupID        int
	Targets        []string
	Ports          []int
	EnableVulnScan bool
}

// Result holds the output of a port scan for a specific IP and port
// It includes banner and parsed service information
// along with the scan timestamp.
type Result struct {
	IP      string
	Port    int
	Status  string // open/closed
	Banner  string
	Service string
	Version string
	CVEs    []string
	Scanned time.Time
}

// Run executes a scanning job across all provided targets and ports in parallel.
// It collects banners and parses them through the plugin system.
func Run(job ScanJob) ([]Result, error) {
	var wg sync.WaitGroup
	results := make([]Result, 0)
	resultChan := make(chan Result, 1000)

	for _, ip := range job.Targets {
		for _, port := range job.Ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				r := scanTarget(ip, port, job.EnableVulnScan)
				resultChan <- r
			}(ip, port)
		}
	}

	// Wait for all scans to finish and close the result channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for res := range resultChan {
		results = append(results, res)
	}

	return results, nil
}

// scanTarget performs a port scan and banner grabbing on a single IP and port.
// It uses service-specific probing and passes banners to the parser for classification.
func scanTarget(ip string, port int, vulnScan bool) Result {
	r := Result{
		IP:      ip,
		Port:    port,
		Status:  "closed",
		Scanned: time.Now(),
	}

	if !ScanPort(ip, port) {
		return r
	}

	r.Status = "open"
	var banner string
	var err error

	if port == 80 || port == 443 || port == 8080 {
		banner, err = HTTPProbe(ip, port)
	} else {
		banner, err = GrabBanner(ip, port)
	}

	if err == nil && banner != "" {
		r.Banner = strings.TrimSpace(banner)
		if info := parser.Dispatch(banner); info != nil {
			r.Service = info.Name
			r.Version = info.Version

			// Optional CVE matching
			if vulnScan {
				ctx := map[string]string{
					strings.ToLower(info.Name) + "/banner": banner,
				}
				matches := plugin.MatchAll(ctx, []int{port}, []string{})
				for _, m := range matches {
					if m.Port == port {
						r.CVEs = m.CVE
						break
					}
				}
			}
		}
	}

	return r
}
