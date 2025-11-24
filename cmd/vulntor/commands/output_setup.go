// Copyright 2025 Vulntor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/vulntor/vulntor/pkg/output"
	"github.com/vulntor/vulntor/pkg/output/subscribers"
)

// setupOutputPipeline creates and configures the output pipeline based on CLI flags.
//
// Flag-based selection:
//   - --output=json: JSONFormatter (structured JSON Lines output to stdout)
//   - --output=text: HumanFormatter (colored tables, human-friendly output)
//   - -v/-vv/-vvv: DiagnosticSubscriber (verbose/debug/trace output to stderr)
//
// Both CE and EE use the same output pipeline - format selection is flag-based,
// not edition-based. This maintains clean separation between business logic
// (which differs by edition) and output rendering (which is identical).
func setupOutputPipeline(cmd *cobra.Command) output.Output {
	stream := output.NewOutputEventStream()

	// Get flags
	outputFormat, _ := cmd.Flags().GetString("output")
	verbosityCount, _ := cmd.Flags().GetCount("verbosity")

	// Format subscriber: --output flag determines Human vs JSON
	if outputFormat == "json" {
		// JSON mode: Structured JSON Lines format (one JSON object per line)
		stream.Subscribe(subscribers.NewJSONFormatter(os.Stdout))
	} else {
		// Human mode: Colored tables, progress bars, human-friendly output
		// Color detection: Check if stdout is a TTY (future enhancement)
		colorEnabled := true // TODO: Auto-detect TTY
		stream.Subscribe(subscribers.NewHumanFormatter(os.Stdout, os.Stderr, colorEnabled))
	}

	// Diagnostic subscriber: Real-time progress messages (always enabled in text mode)
	// Only for text mode (JSON mode should not have styled diagnostic output)
	// Verbosity levels:
	//   - No flags (0): Shows LevelNormal messages (progress updates)
	//   - -v (1): Shows LevelNormal + LevelVerbose (detailed progress)
	//   - -vv (2): Shows up to LevelDebug
	//   - -vvv (3): Shows up to LevelTrace
	if outputFormat != "json" {
		// Always create DiagnosticSubscriber in text mode
		// verbosityCount defaults to 0 (LevelNormal) if no -v flags
		verboseLevel := output.OutputLevel(verbosityCount)
		stream.Subscribe(subscribers.NewDiagnosticSubscriber(verboseLevel, os.Stderr))
	}

	return output.NewDefaultOutput(stream)
}
