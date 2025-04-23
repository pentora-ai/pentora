// cli/serve.go
package cli

import (
	"fmt"
	"net/http"

	"github.com/pentoraai/pentora/pkg/api"
	"github.com/spf13/cobra"
)

// ServeCmd defines the 'serve' command for starting the API server
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Pentora API server",
	Run: func(cmd *cobra.Command, args []string) {
		port := 8080
		fmt.Printf("API server listening on http://localhost:%d\n", port)
		http.ListenAndServe(fmt.Sprintf(":%d", port), api.NewRouter())
	},
}
