// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package dag

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type exportOptions struct {
	output     string
	format     string
	targets    string
	ports      string
	vuln       bool
	noDiscover bool
}

func newExportCommand() *cobra.Command {
	opts := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export programmatic DAG to YAML/JSON",
		Long: `Export the internal DAG structure for a scan configuration.

This command creates a DAG definition based on scan parameters (targets, ports, etc.)
and exports it to YAML or JSON format. Useful for:
- Understanding the internal execution plan
- Creating custom DAG definitions
- Debugging module dependencies
- Documentation and learning`,
		Example: `  # Export to stdout (YAML)
  pentora dag export --targets 192.168.1.1

  # Export to file
  pentora dag export --targets 10.0.0.0/24 --output scan.yaml

  # Export with vulnerability checks enabled
  pentora dag export --targets 192.168.1.1 --vuln --output full-scan.yaml

  # Export as JSON
  pentora dag export --targets 192.168.1.1 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output file (default: stdout)")
	cmd.Flags().StringVar(&opts.format, "format", "yaml", "Output format (yaml|json)")
	cmd.Flags().StringVar(&opts.targets, "targets", "192.168.1.1", "Target hosts for scan")
	cmd.Flags().StringVar(&opts.ports, "ports", "22,80,443", "Ports to scan")
	cmd.Flags().BoolVar(&opts.vuln, "vuln", false, "Include vulnerability evaluation modules")
	cmd.Flags().BoolVar(&opts.noDiscover, "no-discover", false, "Skip discovery modules")

	return cmd
}

func runExport(cmd *cobra.Command, opts *exportOptions) error {
	// Create a simple example DAG for now
	// TODO: In future, integrate with actual scan planner to generate real DAG
	dag := createExampleDAG(opts)

	// Validate the generated DAG
	result := dag.Validate()
	if !result.IsValid() {
		return fmt.Errorf("generated DAG is invalid:\n%s", result.String())
	}

	// Marshal to requested format
	var data []byte
	var err error

	switch opts.format {
	case "yaml":
		data, err = yaml.Marshal(dag)
	case "json":
		data, err = json.MarshalIndent(dag, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s (use yaml or json)", opts.format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal DAG: %w", err)
	}

	// Write to file or stdout
	if opts.output == "" {
		fmt.Print(string(data))
	} else {
		if err := os.WriteFile(opts.output, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("DAG exported to: %s\n", opts.output)
	}

	return nil
}

// createExampleDAG creates a sample DAG based on scan options
// TODO: Replace this with actual integration to pkg/scanexec planner
func createExampleDAG(opts *exportOptions) *engine.DAGSchema {
	dag := &engine.DAGSchema{
		Name:        "scan-dag",
		Description: fmt.Sprintf("Scan DAG for targets: %s", opts.targets),
		Version:     "1.0",
		Nodes:       []engine.DAGNode{},
	}

	// Add config loader node (always present)
	dag.Nodes = append(dag.Nodes, engine.DAGNode{
		ID:     "config-loader",
		Module: "config",
		Produces: []string{
			"config.targets",
			"config.ports",
		},
		Config: map[string]interface{}{
			"targets": opts.targets,
			"ports":   opts.ports,
		},
	})

	// Add discovery node (unless --no-discover)
	if !opts.noDiscover {
		dag.Nodes = append(dag.Nodes, engine.DAGNode{
			ID:     "discovery-tcp",
			Module: "discovery-tcp",
			DependsOn: []string{
				"config-loader",
			},
			Consumes: []string{
				"config.targets",
			},
			Produces: []string{
				"discovery.live_hosts",
			},
		})
	}

	// Add port scan node
	portScanNode := engine.DAGNode{
		ID:     "port-scan",
		Module: "port-scan",
		DependsOn: []string{
			"config-loader",
		},
		Consumes: []string{
			"config.ports",
		},
		Produces: []string{
			"scan.open_ports",
		},
	}

	if !opts.noDiscover {
		portScanNode.DependsOn = append(portScanNode.DependsOn, "discovery-tcp")
		portScanNode.Consumes = append(portScanNode.Consumes, "discovery.live_hosts")
	} else {
		portScanNode.Consumes = append(portScanNode.Consumes, "config.targets")
	}

	dag.Nodes = append(dag.Nodes, portScanNode)

	// Add banner grab node
	dag.Nodes = append(dag.Nodes, engine.DAGNode{
		ID:     "banner-grab",
		Module: "banner-grab",
		DependsOn: []string{
			"port-scan",
		},
		Consumes: []string{
			"scan.open_ports",
		},
		Produces: []string{
			"scan.banners",
		},
	})

	// Add fingerprint node
	dag.Nodes = append(dag.Nodes, engine.DAGNode{
		ID:     "fingerprint",
		Module: "fingerprint",
		DependsOn: []string{
			"banner-grab",
		},
		Consumes: []string{
			"scan.banners",
		},
		Produces: []string{
			"fingerprint.services",
		},
	})

	// Add vuln evaluation node (if --vuln)
	if opts.vuln {
		dag.Nodes = append(dag.Nodes, engine.DAGNode{
			ID:     "vuln-eval",
			Module: "vuln-eval",
			DependsOn: []string{
				"fingerprint",
			},
			Consumes: []string{
				"fingerprint.services",
			},
			Produces: []string{
				"vuln.findings",
			},
		})
	}

	// Add report node (always last)
	reportNode := engine.DAGNode{
		ID:     "report",
		Module: "report",
		DependsOn: []string{
			"fingerprint",
		},
		Consumes: []string{
			"fingerprint.services",
		},
	}

	if opts.vuln {
		reportNode.DependsOn = append(reportNode.DependsOn, "vuln-eval")
		reportNode.Consumes = append(reportNode.Consumes, "vuln.findings")
	}

	dag.Nodes = append(dag.Nodes, reportNode)

	return dag
}
