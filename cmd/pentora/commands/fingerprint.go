package commands

import (
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/pentora-ai/pentora/cmd/pentora/internal/bind"
	"github.com/pentora-ai/pentora/cmd/pentora/internal/format"
	"github.com/pentora-ai/pentora/pkg/fingerprint"
	"github.com/pentora-ai/pentora/pkg/fingerprint/catalogsync"
	"github.com/pentora-ai/pentora/pkg/storage"
)

// NewFingerprintCommand wires CLI helpers for fingerprint catalog management.
func NewFingerprintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fingerprint",
		Aliases: []string{"fp"},
		Short:   "Manage fingerprint probe catalogs",
		GroupID: "core",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newFingerprintSyncCommand())

	return cmd
}

func newFingerprintSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync fingerprint probes from a remote or local source",
		RunE: func(cmd *cobra.Command, _ []string) error {
			formatter := format.FromCommand(cmd)
			// Bind flags to options using centralized binder
			opts, err := bind.BindFingerprintOptions(cmd)
			if err != nil {
				return formatter.PrintTotalFailureSummary("sync fingerprint catalog", err, fingerprint.ErrorCode(err))
			}

			destination := opts.CacheDir
			if destination == "" {
				if cfg, ok := storage.ConfigFromContext(cmd.Context()); ok {
					destination = filepath.Join(cfg.WorkspaceRoot, "cache", "fingerprint")
				} else {
					derr := fingerprint.NewStorageDisabledError()
					return formatter.PrintTotalFailureSummary("sync fingerprint catalog", derr, fingerprint.ErrorCode(derr))
				}
			}

			svc := catalogsync.Service{
				CacheDir: destination,
			}

			if opts.FilePath != "" {
				svc.Source = catalogsync.FileSource{Path: opts.FilePath}
			} else {
				svc.Source = catalogsync.HTTPSource{URL: opts.URL}
			}
			svc.Store = catalogsync.FileStore{Path: filepath.Join(destination, "probe.catalog.yaml")}

			catalog, err := svc.Sync(cmd.Context())
			if err != nil {
				wrapped := fingerprint.WrapSyncError(err)
				return formatter.PrintTotalFailureSummary("sync fingerprint catalog", wrapped, fingerprint.ErrorCode(wrapped))
			}

			log.Info().Str("cache", destination).Int("groups", len(catalog.Groups)).Int("probes", totalProbes(catalog)).Msg("fingerprint probes synced")
			return nil
		},
	}

	cmd.Flags().String("file", "", "Load probe catalog from a local file")
	cmd.Flags().String("url", "", "Download probe catalog from a remote URL")
	cmd.Flags().String("cache-dir", "", "Override probe cache destination directory")

	return cmd
}

func totalProbes(catalog *fingerprint.ProbeCatalog) int {
	if catalog == nil {
		return 0
	}
	total := 0
	for _, group := range catalog.Groups {
		total += len(group.Probes)
	}
	return total
}
