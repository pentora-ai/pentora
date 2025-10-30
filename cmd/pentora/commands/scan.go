package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/pkg/appctx"
	"github.com/pentora-ai/pentora/pkg/engine"
	parsepkg "github.com/pentora-ai/pentora/pkg/modules/parse" // Register parse modules if needed
	_ "github.com/pentora-ai/pentora/pkg/modules/reporting"    // Register reporting modules if needed
	_ "github.com/pentora-ai/pentora/pkg/modules/scan"         // Register this module
	"github.com/pentora-ai/pentora/pkg/scanexec"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/pentora-ai/pentora/pkg/stringutil"
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
		logger := log.With().Str("command", "scan").Logger()
		logger.Info().Strs("targets", args).Msg("Initializing scan command")

		// Bind flags to options using centralized binder
		params, err := bind.BindScanOptions(cmd, args)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to bind scan options")
			fmt.Fprintf(os.Stderr, "[ERROR] Invalid scan options: %v\n", err)
			return
		}

		svc := scanexec.NewService()

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

		// Create and attach storage backend for scan result persistence
		storageConfig, err := storage.DefaultConfig()
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to get storage config, scans will not be persisted")
		} else {
			storageBackend, err := storage.NewBackend(orchestratorCtx, storageConfig)
			if err != nil {
				logger.Warn().Err(err).Msg("Failed to create storage backend, scans will not be persisted")
			} else {
				// Initialize storage
				if err := storageBackend.Initialize(orchestratorCtx); err != nil {
					logger.Warn().Err(err).Msg("Failed to initialize storage, scans will not be persisted")
				} else {
					svc = svc.WithStorage(storageBackend)
					logger.Info().Msg("Storage backend initialized for scan persistence")

					// Ensure storage is closed when scan completes
					defer func() {
						if err := storageBackend.Close(); err != nil {
							logger.Warn().Err(err).Msg("Failed to close storage backend")
						}
					}()
				}
			}
		}

		// Enable progress logging if interactive flag is set
		interactive, _ := cmd.Flags().GetBool("progress")
		if interactive {
			svc = svc.WithProgressSink(&progressLogger{logger: logger})
		}

		if params.OutputFormat == "text" {
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
		switch strings.ToLower(params.OutputFormat) {
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
				// Print summary table first
				printScanSummary(res, finalProfiles)
				// Then print detailed results
				printAssetProfileTextOutput(finalProfiles)
			} else {
				fmt.Println("\nScan completed, but no asset profiles were generated.")
			}
		}
	},
}

func printAssetProfileTextOutput(profiles []engine.AssetProfile) {
	fmt.Println("--- Scan Results ---")
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

// printScanSummary displays a human-readable summary table of scan results
func printScanSummary(res *scanexec.Result, profiles []engine.AssetProfile) {
	if res == nil || len(profiles) == 0 {
		return
	}

	// Calculate summary statistics
	hostsFound := len(profiles)
	totalOpenPorts := 0
	totalVulns := 0
	servicesMap := make(map[string]bool) // unique services

	for _, profile := range profiles {
		totalVulns += profile.TotalVulnerabilities
		for _, portList := range profile.OpenPorts {
			totalOpenPorts += len(portList)
			for _, port := range portList {
				if port.Service.Name != "" {
					serviceName := port.Service.Name
					if port.PortNumber > 0 {
						serviceName = fmt.Sprintf("%s (%d)", port.Service.Name, port.PortNumber)
					}
					servicesMap[serviceName] = true
				}
			}
		}
	}

	// Build services string
	var services []string
	for svc := range servicesMap {
		services = append(services, svc)
	}
	sort.Strings(services)
	servicesStr := strings.Join(services, ", ")

	// Calculate duration
	var duration string
	if res.StartTime != "" && res.EndTime != "" {
		startTime, errStart := time.Parse(time.RFC3339Nano, res.StartTime)
		endTime, errEnd := time.Parse(time.RFC3339Nano, res.EndTime)
		if errStart == nil && errEnd == nil {
			durationTime := endTime.Sub(startTime)
			duration = fmt.Sprintf("%.1fs", durationTime.Seconds())
		} else {
			duration = "N/A"
		}
	} else {
		duration = "N/A"
	}

	// Get primary target
	target := "N/A"
	if len(profiles) > 0 {
		target = profiles[0].Target
	}

	// Print summary table
	separator := "════════════════════════════════════════════════════"
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("%-15s %s\n", "Target:", target)
	fmt.Printf("%-15s %s\n", "Duration:", duration)
	fmt.Printf("%-15s %d\n", "Hosts Found:", hostsFound)
	fmt.Printf("%-15s %d\n", "Open Ports:", totalOpenPorts)
	// Only show Services line if any services were detected
	if servicesStr != "" {
		fmt.Printf("%-15s %s\n", "Services:", servicesStr)
	}
	fmt.Printf("\n%-15s %d\n", "Vulnerabilities:", totalVulns)
	fmt.Printf("%s\n\n", separator)
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
	ScanCmd.Flags().StringP("ports", "p", "", "Ports/port ranges for TCP scan (e.g., 'top-1000', '22,80,443', '1-65535')")
	ScanCmd.Flags().String("profile", "", "Predefined scan profile (e.g., 'quick_discovery', 'full_vuln_scan')")
	ScanCmd.Flags().String("level", "default", "Scan intensity level (e.g., 'light', 'default', 'comprehensive', 'intrusive')")
	ScanCmd.Flags().StringSlice("tags", []string{}, "Only include modules with these tags (comma-separated)")
	ScanCmd.Flags().StringSlice("exclude-tags", []string{}, "Exclude modules with these tags (comma-separated)")
	ScanCmd.Flags().Bool("vuln", false, "Enable vulnerability assessment modules (shortcut for a common intent)")
	ScanCmd.Flags().Bool("only-discover", false, "Run only discovery modules (scan and vuln phases are skipped)")
	ScanCmd.Flags().Bool("no-discover", false, "Skip discovery phase and proceed directly to port scanning/vuln")
	ScanCmd.Flags().Bool("progress", false, "Print live progress updates during the scan")
	ScanCmd.Flags().String("fingerprint-cache", "", "Path to fingerprint catalog cache directory")
	ScanCmd.Flags().StringP("output", "o", "text", "Output format: text, json, yaml")
	ScanCmd.Flags().String("timeout", "1s", "Default timeout for network operations like ping/port connect")
	ScanCmd.Flags().Int("concurrency", 50, "Default concurrency for parallel operations")

	// Ping specific flags - planner can use these if ICMP module is selected
	ScanCmd.Flags().Bool("ping", true, "Enable ICMP host discovery (default: true)")
	ScanCmd.Flags().Int("ping-count", 1, "Number of ICMP pings per host")
	ScanCmd.Flags().Bool("allow-loopback", false, "Allow scanning loopback addresses")
}
