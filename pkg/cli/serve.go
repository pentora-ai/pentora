// cli/serve.go
package cli

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pentora-ai/pentora/pkg/api/dashboard"
	"github.com/spf13/cobra"
)

// ServeCmd defines the 'serve' command for starting the API server
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Pentora API server",
	Run: func(cmd *cobra.Command, args []string) {
		port := 8080

		mux := http.NewServeMux()

		// Frontend UI register
		dashboard.RegisterFrontend(mux)

		// Buraya ileride API endpoint'leri de eklenebilir
		// mux.HandleFunc("/api/...", api.SomeHandler)

		// API endpoint register
		// mux.HandleFunc("/api/version", version.VersionHandler)

		log.Printf("🚀 Pentora UI serving at http://localhost:%d\n", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	},
}
