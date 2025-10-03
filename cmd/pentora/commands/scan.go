package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/engine"
	parsepkg "github.com/pentora-ai/pentora/pkg/modules/parse" // Register parse modules if needed
	_ "github.com/pentora-ai/pentora/pkg/modules/reporting"    // Register reporting modules if needed
	_ "github.com/pentora-ai/pentora/pkg/modules/scan"         // Register this module
	"github.com/pentora-ai/pentora/pkg/scanexec"
	"github.com/pentora-ai/pentora/pkg/stringutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Flags for the scan command (remains the same)
var (
	// scanTargets       []string
	scanPorts         string
	scanProfile       string
	scanLevel         string
	scanIncludeTags   []string
	scanExcludeTags   []string
	scanEnableVuln    bool
	scanOutputFormat  string
	scanCustomTimeout string
	scanConcurrency   int  // This might become a global config or part of ScanIntent for planner
	scanEnablePing    bool // This now feeds into ScanIntent
	scanPingCount     int  // This now feeds into ScanIntent
	scanAllowLoopback bool // This now feeds into ScanIntent
	scanOnlyDiscover  bool
	scanSkipDiscover  bool
	scanInteractive   bool
)

// ScanCmd defines the 'scan' command for comprehensive scanning.
var ScanCmd = &cobra.Command{
	Use:   "scan [targets...]",
	Short: "Perform a comprehensive scan on specified targets",
	Long: `Performs various scanning stages based on selected profile, level, or flags.
The command automatically plans the execution DAG using available modules.`,
	GroupID: "scan",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scanTargetsFromCLI := args

		logger := log.With().Str("command", "scan").Logger()
		logger.Info().Strs("targets", scanTargetsFromCLI).Msg("Initializing scan command")

		if scanOnlyDiscover && scanSkipDiscover {
			fmt.Fprintln(os.Stderr, "[ERROR] --only-discover and --no-discover cannot be used together")
			return
		}
		if scanOnlyDiscover {
			scanEnableVuln = false
		}

		svc := scanexec.NewService()
		if scanInteractive {
			svc = svc.WithProgressSink(&progressLogger{logger: logger})
		}

		ctxFromCmd := cmd.Context()
		if ctxFromCmd == nil && cmd.Root() != nil {
			ctxFromCmd = cmd.Root().Context()
		}
		appMgr, ok := ctxFromCmd.Value(engine.AppManagerKey).(*engine.AppManager)
		if !ok || appMgr == nil {
			logger.Error().Msg("AppManager not found in context.")
			fmt.Fprintln(os.Stderr, "[ERROR] Critical: AppManager not found in context.")
			return
		}
		orchestratorCtx := context.WithValue(appMgr.Context(), engine.AppManagerKey, appMgr)
		orchestratorCtx = appctx.WithConfig(orchestratorCtx, appMgr.Config())

		params := scanexec.Params{
			Targets:       scanTargetsFromCLI,
			Profile:       scanProfile,
			Level:         scanLevel,
			IncludeTags:   scanIncludeTags,
			ExcludeTags:   scanExcludeTags,
			EnableVuln:    scanEnableVuln,
			Ports:         scanPorts,
			CustomTimeout: scanCustomTimeout,
			EnablePing:    scanEnablePing,
			PingCount:     scanPingCount,
			AllowLoopback: scanAllowLoopback,
			Concurrency:   scanConcurrency,
			OutputFormat:  scanOutputFormat,
			RawInputs: map[string]interface{}{
				"config.scan.requested_profile": scanProfile,
			},
			OnlyDiscover: scanOnlyDiscover,
			SkipDiscover: scanSkipDiscover,
		}

		if scanOutputFormat == "text" {
			logger.Info().Msg("Starting scan execution with automatically planned DAG...")
		}

		res, runErr := svc.Run(orchestratorCtx, params)
		if runErr != nil {
			logger.Error().Err(runErr).Msg("Scan execution failed")
		}

		finalDataContext := map[string]interface{}{}
		if res != nil && res.RawContext != nil {
			finalDataContext = res.RawContext
		}

		executionErr := runErr
		assetProfileDataKey := "asset.profiles"

		// 2. AssetProfile verisini DataContext'ten al ve cast et
		var finalProfiles []engine.AssetProfile
		if rawProfiles, found := finalDataContext[assetProfileDataKey]; found {
			// DataContext, modül çıktılarını []interface{} listesi olarak saklar.
			// AssetProfileBuilder tek bir çıktı (bir []AssetProfile listesi) ürettiği için,
			// DataContext'teki liste tek elemanlıdır: []interface{}{ []engine.AssetProfile{...} }
			if profileList, listOk := rawProfiles.([]interface{}); listOk && len(profileList) > 0 {
				if castedProfiles, castOk := profileList[0].([]engine.AssetProfile); castOk {
					finalProfiles = castedProfiles
				} else if executionErr == nil {
					executionErr = fmt.Errorf("could not cast asset profile data to expected type: %T", profileList[0])
				}
			} else if rawProfiles != nil && executionErr == nil {
				executionErr = fmt.Errorf("asset profile data has unexpected type: %T", rawProfiles)
			}
		} else if executionErr == nil {
			logger.Info().Msg("No 'asset.profiles' data found in scan results.")
			executionErr = fmt.Errorf("scan completed, but no asset profile data was generated")
		}

		// 3. Seçilen formata göre çıktıyı yazdır
		switch strings.ToLower(scanOutputFormat) {
		case "json":
			// Eğer hiç profil yoksa ama hata da yoksa boş bir liste yazdır
			if finalProfiles == nil {
				finalProfiles = []engine.AssetProfile{}
			}
			jsonData, jsonErr := json.MarshalIndent(finalProfiles, "", "  ")
			if jsonErr != nil {
				logger.Error().Err(jsonErr).Msg("Failed to marshal AssetProfile to JSON")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to generate JSON output: %v\n", jsonErr)
			} else {
				fmt.Println(string(jsonData))
			}
		case "yaml":
			if finalProfiles == nil {
				finalProfiles = []engine.AssetProfile{}
			}
			yamlData, yamlErr := yaml.Marshal(finalProfiles)
			if yamlErr != nil {
				logger.Error().Err(yamlErr).Msg("Failed to marshal AssetProfile to YAML")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to generate YAML output: %v\n", yamlErr)
			} else {
				fmt.Println(string(yamlData))
			}
		default: // "text"
			if executionErr != nil {
				fmt.Fprintf(os.Stderr, "\nScan finished with errors: %v\n", executionErr)
			}
			if len(finalProfiles) > 0 {
				printAssetProfileTextOutput(finalProfiles)
			} else {
				fmt.Println("\nScan completed, but no asset profiles were generated.")
			}
		}
	},
}

func printAssetProfileTextOutput(profiles []engine.AssetProfile) {
	spew.Dump(profiles[0].OpenPorts) // --- IGNORE ---
	fmt.Println("\n--- Scan Results ---")
	for _, asset := range profiles {
		fmt.Printf("\n## Target: %s (IPs: %v)\n", asset.Target, getMapKeys(asset.ResolvedIPs))
		fmt.Printf("   Is Alive: %v\n", asset.IsAlive)
		if len(asset.Hostnames) > 0 {
			fmt.Printf("   Hostnames: %v\n", asset.Hostnames)
		}

		if len(asset.OpenPorts) > 0 {
			fmt.Println("   --- Open Ports ---")
			// Portları sıralı göstermek için IP'leri sırala
			var sortedIPs []string
			for ip := range asset.OpenPorts {
				sortedIPs = append(sortedIPs, ip)
			}
			sort.Strings(sortedIPs)

			for _, ip := range sortedIPs {
				fmt.Printf("     IP: %s\n", ip)
				// Portları sıralı göstermek için port numarasına göre sırala
				portProfiles := asset.OpenPorts[ip]
				sort.Slice(portProfiles, func(i, j int) bool {
					return portProfiles[i].PortNumber < portProfiles[j].PortNumber
				})

				for _, port := range portProfiles {
					fmt.Printf("       - Port: %d/%s (%s)\n", port.PortNumber, port.Protocol, port.Status)
					if port.Service.Name != "" || port.Service.Product != "" {
						fmt.Printf("         Service: %s %s %s\n", port.Service.Name, port.Service.Product, port.Service.Version)
					}
					if port.Service.RawBanner != "" {
						fmt.Printf("         Banner: %s\n", stringutil.Ellipsis(port.Service.RawBanner, 80))
					}
					if port.Service.ParsedAttributes != nil {
						attrs := port.Service.ParsedAttributes
						printedFingerprintList := false
						if rawMatches, ok := attrs["fingerprints"]; ok {
							if matches, ok := rawMatches.([]parsepkg.FingerprintParsedInfo); ok && len(matches) > 0 {
								printedFingerprintList = true
								fmt.Println("         Fingerprints:")
								for _, match := range matches {
									fmt.Printf("           - %s", match.Product)
									if match.Version != "" {
										fmt.Printf(" %s", match.Version)
									}
									if match.Vendor != "" {
										fmt.Printf(" [%s]", match.Vendor)
									}
									fmt.Printf(" (confidence %.2f", match.Confidence)
									if match.SourceProbe != "" {
										fmt.Printf(", probe %s", match.SourceProbe)
									}
									fmt.Println(")")
									if match.CPE != "" {
										fmt.Printf("             CPE: %s\n", match.CPE)
									}
									if match.Description != "" {
										fmt.Printf("             Notes: %s\n", match.Description)
									}
								}
							}
						}
						if !printedFingerprintList {
							if confidence, ok := attrs["fingerprint_confidence"]; ok {
								fmt.Printf("         Fingerprint Confidence: %v\n", confidence)
							}
							if vendor, ok := attrs["vendor"]; ok {
								fmt.Printf("         Vendor: %v\n", vendor)
							}
							if cpe, ok := attrs["cpe"]; ok {
								fmt.Printf("         CPE: %v\n", cpe)
							}
						}
					}

					if len(port.Vulnerabilities) > 0 {
						fmt.Println("         Vulnerabilities:")
						for _, vuln := range port.Vulnerabilities {
							fmt.Printf("           - [%s] %s (%s)\n", vuln.Severity, vuln.ID, vuln.Summary)
						}
					}
				}
			}
		} else {
			fmt.Println("   No open ports found.")
		}
	}
	fmt.Println("\n--- End of Scan Results ---")
}

// Helper function to get keys from a map for printing.
func getMapKeys(m map[string]time.Time) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type progressLogger struct {
	logger zerolog.Logger
}

func (p *progressLogger) OnEvent(ev scanexec.ProgressEvent) {
	entry := p.logger.Info().
		Str("phase", ev.Phase).
		Str("module", ev.Module).
		Str("status", ev.Status)
	if ev.ModuleID != "" {
		entry = entry.Str("module_id", ev.ModuleID)
	}
	if ev.Message != "" {
		entry = entry.Str("message", ev.Message)
	}
	entry.Msg("scan progress")
}

func init() {
	// Flags for ScanCmd (ensure these are descriptive for the planner)
	ScanCmd.Flags().StringVarP(&scanPorts, "ports", "p", "", "Ports/port ranges for TCP scan (e.g., 'top-1000', '22,80,443', '1-65535')")
	ScanCmd.Flags().StringVar(&scanProfile, "profile", "", "Predefined scan profile (e.g., 'quick_discovery', 'full_vuln_scan')")
	ScanCmd.Flags().StringVar(&scanLevel, "level", "default", "Scan intensity level (e.g., 'light', 'default', 'comprehensive', 'intrusive')")
	ScanCmd.Flags().StringSliceVar(&scanIncludeTags, "tags", []string{}, "Only include modules with these tags (comma-separated)")
	ScanCmd.Flags().StringSliceVar(&scanExcludeTags, "exclude-tags", []string{}, "Exclude modules with these tags (comma-separated)")
	ScanCmd.Flags().BoolVar(&scanEnableVuln, "vuln", false, "Enable vulnerability assessment modules (shortcut for a common intent)")
	ScanCmd.Flags().BoolVar(&scanOnlyDiscover, "only-discover", false, "Run only discovery modules (scan and vuln phases are skipped)")
	ScanCmd.Flags().BoolVar(&scanSkipDiscover, "no-discover", false, "Skip discovery phase and proceed directly to port scanning/vuln")
	ScanCmd.Flags().BoolVar(&scanInteractive, "progress", false, "Print live progress updates during the scan")
	ScanCmd.Flags().String("fingerprint-cache", "", "Path to fingerprint catalog cache directory")
	ScanCmd.Flags().StringVarP(&scanOutputFormat, "output", "o", "text", "Output format: text, json, yaml")
	ScanCmd.Flags().StringVar(&scanCustomTimeout, "timeout", "1s", "Default timeout for network operations like ping/port connect")
	ScanCmd.Flags().IntVar(&scanConcurrency, "concurrency", 50, "Default concurrency for parallel operations")

	// Ping specific flags - planner can use these if ICMP module is selected
	ScanCmd.Flags().BoolVar(&scanEnablePing, "ping", true, "Enable ICMP host discovery (default: true)")
	ScanCmd.Flags().IntVar(&scanPingCount, "ping-count", 1, "Number of ICMP pings per host")
	ScanCmd.Flags().BoolVar(&scanAllowLoopback, "allow-loopback", false, "Allow scanning loopback addresses")
}
