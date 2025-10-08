package commands

import (
	"fmt"
	"time"

	"github.com/pentora-ai/pentora/pkg/scanexec"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// NewProgressDemoCommand wires a simple demo showcasing the progress UI without running a real scan.
func NewProgressDemoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress-demo",
		Short: "Showcase the interactive progress UI with simulated events",
		RunE: func(cmd *cobra.Command, args []string) error {
			sink, err := newProgressUISink()
			if err != nil {
				return fmt.Errorf("progress UI unavailable: %w", err)
			}

			finalStatus := "completed"
			defer sink.Stop(finalStatus, nil)

			steps := []struct {
				delay   time.Duration
				event   scanexec.ProgressEvent
				logFunc func()
			}{
				{
					delay: 300 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "plan",
						Module:    "agent:init",
						Status:    "start",
						Message:   "receiving user request",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Msg("AI agent → request received: \"Scan the critical network segment\"") },
				},
				{
					delay: 450 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "plan",
						Module:    "agent:planner",
						Status:    "running",
						Message:   "enumerating targets and constraints",
						Timestamp: time.Now(),
					},
					logFunc: func() {
						log.Info().Strs("targets", []string{"10.1.0.0/24", "10.2.12.7"}).Msg("Planner assessing scope")
					},
				},
				{
					delay: 400 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "plan",
						Module:    "agent:planner",
						Status:    "running",
						Message:   "resolving module graph",
						Timestamp: time.Now(),
					},
					logFunc: func() {
						log.Debug().Msg("Planner shortlisted modules: tcp-port-discovery, banner-grabber, fingerprint-parser")
					},
				},
				{
					delay: 500 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "plan",
						Module:    "agent:planner",
						Status:    "completed",
						Message:   "plan ready (3 stages)",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Int("stages", 3).Msg("Plan finalised") },
				},
				{
					delay: 350 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "agent:executor",
						Status:    "running",
						Message:   "launching tcp-port-discovery",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Msg("Executor → probing ports with concurrency=100") },
				},
				{
					delay: 350 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:tcp-port-discovery",
						Status:    "completed",
						Message:   "18 open ports discovered",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Int("open_ports", 18).Msg("Discovery stage finished") },
				},
				{
					delay: 400 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:banner-grabber",
						Status:    "running",
						Message:   "collecting service banners",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Msg("Banner grabber dispatching probes") },
				},
				{
					delay: 400 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:banner-grabber",
						Status:    "failed",
						Message:   "probe timeout on 10.1.0.42:443",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Warn().Str("target", "10.1.0.42:443").Msg("Probe timed out, retrying with TLS shim") },
				},
				{
					delay: 450 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:banner-grabber",
						Status:    "completed",
						Message:   "banners collected",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Msg("Banner grabber recovered after retry") },
				},
				{
					delay: 400 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:fingerprint-parser",
						Status:    "running",
						Message:   "matching fingerprints",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Msg("Fingerprint parser analysing banners") },
				},
				{
					delay: 350 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "module:fingerprint-parser",
						Status:    "completed",
						Message:   "found 4 confirmed services",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Info().Int("services", 4).Msg("Fingerprinting completed") },
				},
				{
					delay: 450 * time.Millisecond,
					event: scanexec.ProgressEvent{
						Phase:     "run",
						Module:    "agent:executor",
						Status:    "running",
						Message:   "assembling asset profile",
						Timestamp: time.Now(),
					},
					logFunc: func() { log.Debug().Msg("Executor stitching asset profile context") },
				},
			}

			for _, step := range steps {
				sink.OnEvent(step.event)
				if step.logFunc != nil {
					step.logFunc()
				}
				time.Sleep(step.delay)
			}

			sink.OnEvent(scanexec.ProgressEvent{
				Phase:     "run",
				Module:    "agent:executor",
				Status:    "completed",
				Message:   "workflow finished",
				Timestamp: time.Now(),
			})
			log.Info().Msg("AI agent → mission complete. Report saved to workspace.")
			// finalStatus = "completed"
			time.Sleep(600 * time.Millisecond)

			return nil
		},
	}

	return cmd
}
