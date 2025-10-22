package storage

import (
	"fmt"

	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newGCCommand() *cobra.Command {
	var (
		dryRun     bool
		orgID      string
		maxScans   int
		maxAgeDays int
	)

	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Run garbage collection on stored scans",
		Long: `Run garbage collection to clean up old scans based on retention policies.

Retention policies define how long scans are kept before being deleted:
  - max_scans: Maximum number of scans to retain (oldest deleted first)
  - max_age_days: Maximum age of scans in days (older scans deleted)

Both policies can be active simultaneously. Scans are deleted if they violate
EITHER condition (OR logic).

The command reads retention policies from:
  1. Command flags (--max-scans, --max-age-days) - highest priority
  2. Config file (storage.retention.max_scans, storage.retention.max_age_days)
  3. Environment variables (PENTORA_RETENTION_MAX_SCANS, PENTORA_RETENTION_MAX_AGE_DAYS)

Use --dry-run to preview which scans would be deleted without actually deleting them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Create storage backend
			storageConfig, err := storage.DefaultConfig()
			if err != nil {
				return fmt.Errorf("get storage config: %w", err)
			}

			// Apply retention policy overrides from flags
			if maxScans > 0 || maxAgeDays > 0 {
				// User provided flags - override config
				storageConfig.Retention = storage.RetentionConfig{
					MaxScans:   maxScans,
					MaxAgeDays: maxAgeDays,
				}
			}

			// Validate config
			if err := storageConfig.Validate(); err != nil {
				return fmt.Errorf("invalid storage config: %w", err)
			}

			backend, err := storage.NewBackend(ctx, storageConfig)
			if err != nil {
				return fmt.Errorf("create storage backend: %w", err)
			}
			defer func() {
				if err := backend.Close(); err != nil {
					log.Warn().Err(err).Msg("Failed to close storage backend")
				}
			}()

			// Check if retention policy is enabled
			if !storageConfig.Retention.IsEnabled() {
				log.Warn().Msg("No retention policy configured. Use --max-scans or --max-age-days to enable GC.")
				fmt.Println("No retention policy configured. Nothing to clean up.")
				fmt.Println()
				fmt.Println("To enable garbage collection, set a retention policy:")
				fmt.Println("  pentora storage gc --max-scans=100")
				fmt.Println("  pentora storage gc --max-age-days=30")
				fmt.Println("  pentora storage gc --max-scans=100 --max-age-days=30")
				return nil
			}

			// Log retention policy
			log.Info().
				Int("max_scans", storageConfig.Retention.MaxScans).
				Int("max_age_days", storageConfig.Retention.MaxAgeDays).
				Bool("dry_run", dryRun).
				Str("org_id", orgID).
				Msg("Starting garbage collection")

			if dryRun {
				fmt.Println("DRY RUN MODE - No scans will be deleted")
				fmt.Println()
			}

			// Print retention policy
			fmt.Printf("Retention Policy:\n")
			if storageConfig.Retention.MaxScans > 0 {
				fmt.Printf("  - Keep maximum %d scans (oldest deleted first)\n", storageConfig.Retention.MaxScans)
			}
			if storageConfig.Retention.MaxAgeDays > 0 {
				fmt.Printf("  - Keep scans from last %d days (older scans deleted)\n", storageConfig.Retention.MaxAgeDays)
			}
			fmt.Println()

			// Run GC
			result, err := backend.GarbageCollect(ctx, storage.GCOptions{
				DryRun: dryRun,
				OrgID:  orgID,
			})
			if err != nil {
				return fmt.Errorf("garbage collection failed: %w", err)
			}

			// Print results
			if result.ScansDeleted == 0 {
				fmt.Println("No scans to clean up")
			} else {
				if dryRun {
					fmt.Printf("Would delete %d scan(s):\n", result.ScansDeleted)
				} else {
					fmt.Printf("Deleted %d scan(s):\n", result.ScansDeleted)
				}
				for _, scanID := range result.DeletedScanIDs {
					fmt.Printf("  - %s\n", scanID)
				}
			}

			// Print errors if any
			if len(result.Errors) > 0 {
				fmt.Printf("\nEncountered %d error(s):\n", len(result.Errors))
				for i, err := range result.Errors {
					fmt.Printf("  %d. %v\n", i+1, err)
				}
			}

			if dryRun && result.ScansDeleted > 0 {
				fmt.Println()
				fmt.Println("To actually delete these scans, run without --dry-run:")
				fmt.Println("  pentora storage gc")
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview scans to be deleted without actually deleting them")
	cmd.Flags().StringVar(&orgID, "org-id", "", "Organization ID to clean up (default: all orgs)")
	cmd.Flags().IntVar(&maxScans, "max-scans", 0, "Maximum number of scans to retain (0 = no limit)")
	cmd.Flags().IntVar(&maxAgeDays, "max-age-days", 0, "Maximum age of scans in days (0 = no limit)")

	return cmd
}
