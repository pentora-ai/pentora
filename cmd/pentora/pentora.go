// cmd/pentora/pentora.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/pentora-ai/pentora/cmd"
	"github.com/pentora-ai/pentora/cmd/license"
	cmdVersion "github.com/pentora-ai/pentora/cmd/version"
	"github.com/pentora-ai/pentora/pkg/cli"
	"github.com/pentora-ai/pentora/pkg/config/static"
	"github.com/pentora-ai/pentora/pkg/core"
	lic "github.com/pentora-ai/pentora/pkg/license"
	"github.com/pentora-ai/pentora/pkg/safe"
	"github.com/pentora-ai/pentora/pkg/server"
	"github.com/pentora-ai/pentora/pkg/server/service"
	"github.com/pentora-ai/pentora/pkg/version"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"

	//"github.com/pentora-ai/pentora/pkg/core/logger"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pentora",
	Short: "Pentora - Platform-independent vulnerability scanner",
	Long:  `Pentora is a cross-platform security scanner designed to find vulnerabilities and misconfigurations in your infrastructure.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Pentora CLI. Use --help for available commands.")
	},
}

func loadGlobalLicense() {
	lic.GlobalStatus = lic.Check(lic.GetDefaultLicensePath(), lic.GetPublicKeyPath())

	if lic.GlobalStatus.Valid {
		fmt.Println("🔐 License OK –", lic.GlobalStatus.Payload.Licensee)
	} else if lic.GlobalStatus.Error != nil {
		fmt.Println("⚠️ License error:", lic.GlobalStatus.Error)
	} else {
		fmt.Println("⚠️ No license found. Running in free mode.")
	}
}

func init() {
	rootCmd.AddCommand(cli.ServeCmd)
	rootCmd.AddCommand(cli.ScanCmd)
	rootCmd.AddCommand(license.LicenseCmd)
	rootCmd.AddCommand(cmdVersion.VersionCmd)
}

func main() {
	// pentora config inits
	pConfig := cmd.NewPentoraConfiguration()

	runCmd(&pConfig.Configuration)

	logrus.Exit(0)
}

func runCmd(staticConfiguration *static.Configuration) error {
	if err := core.SetupLogger(staticConfiguration); err != nil {
		return fmt.Errorf("failed to setting up logger: %w", err)
	}

	app := core.NewAppManager()
	_ = app.Init()

	app.HookManager.Register("plugin:beforeRun:ping", func(ctx context.Context) {
		log.Info().Msg("Running before plugin:ping hooks...")
	})

	app.HookManager.Register("plugin:onError:ping", func(ctx context.Context) {
		log.Info().Msg("Running onError plugin:ping hooks...")
	})

	app.HookManager.Register("plugin:afterRun:ping", func(ctx context.Context) {
		log.Info().Msg("Running after plugin:ping hooks...")
	})

	err := app.Orchestrator.RunPluginsDAGParallelLayers(app.Context(), "192.168.1.1")
	if err != nil {
		log.Err(err).Msg("scan failed:")
	}

	app.HookManager.Register("onShutdown", func(ctx context.Context) {
		log.Info().Msg("Running shutdown hooks...")
	})

	log.Info().Str("version", version.Version).
		Msgf("Pentora version %s built on %s", version.Version, version.BuildDate)

	jsonConf, err := json.Marshal(staticConfiguration)
	if err != nil {
		log.Error().Err(err).Msg("Could not marshal static configuration")
		log.Debug().Interface("staticConfiguration", staticConfiguration).Msg("Static configuration loaded [struct]")
	} else {
		log.Debug().RawJSON("staticConfiguration", jsonConf).Msg("Static configuration loaded [json]")
	}

	//
	if staticConfiguration.Global.CheckNewVersion {
		checkNewVersion()
	}

	srv, err := setupServer(staticConfiguration)
	if err != nil {
		return err
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	//if staticConfiguration.Ping != nil {
	//	staticConfiguration.Ping.WithContext(ctx)
	//}

	srv.Start(ctx)
	defer srv.Close()

	sent, err := daemon.SdNotify(false, "READY=1")
	if !sent && err != nil {
		log.Error().Err(err).Msg("Failed to notify systemd")
	}

	srv.Wait()
	app.Shutdown()
	log.Info().Msg("Shutting down Pentora...")

	return nil
}

func setupServer(staticConfiguration *static.Configuration) (*server.Server, error) {

	ctx := context.Background()
	routinesPool := safe.NewPool(ctx)

	service.NewManagerFactory(*staticConfiguration)

	return server.NewServer(routinesPool), nil
}

// checkNewVersion periodically checks for a new version of the application.
// It starts a goroutine that waits for 10 minutes initially and then checks
// for a new version every 24 hours using the version.CheckNewVersion function.
func checkNewVersion() {
	ticker := time.Tick(24 * time.Hour)
	safe.Go(func() {
		for time.Sleep(10 * time.Minute); ; <-ticker {
			version.CheckNewVersion()
		}
	})
}
