// pkg/api/dashboard/dashboard.go
// Package dashboard provides the API for the Pentora dashboard.
package dashboard

import (
	"io/fs"
	"net/http"

	"github.com/containous/mux"
	"github.com/pentora-ai/pentora/ui"
)

// Handler expose dashboard routes.
type Handler struct {
	BasePath string
}

// commonHeaders is a middleware function that adds common HTTP headers to the response.
// It sets the Content-Security-Policy header to allow iframe embedding only from the
// specified domains ('self', https://pentora.ai, and https://*.pentora.ai). This helps
// enhance security by restricting the sources from which iframes can be loaded.
//
// Parameters:
//
//	next - The next http.Handler in the middleware chain.
//
// Returns:
//
//	An http.Handler that wraps the provided handler and adds the specified headers.
func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow iframes from pentora domains only
		w.Header().Set("Content-Security-Policy", "frame-src 'self' https://pentora.ai https://*.pentora.ai;")

		// w.Header().Set("X-Custom-Header", "value")

		next.ServeHTTP(w, r)
	})
}

// RegisterFrontend registers the frontend routes for serving static assets and
// handling Single Page Application (SPA) requests.
//
// It sets up two handlers:
//  1. A static asset handler that serves files from the "dist" directory under
//     the "/assets/" URL path.
//  2. An SPA handler that serves the "index.html" file for all other requests,
//     enabling client-side routing.
//
// Parameters:
//   - mux: The HTTP request multiplexer to which the handlers will be registered.
//
// Panics:
//
//	This function will panic if it fails to create a sub-filesystem from the
//	"dist" directory.
func RegisterFrontend(mux *http.ServeMux) {
	// Make the files under the "dist" directory the root FS
	fsys, err := fs.Sub(ui.Assets, "dist")
	if err != nil {
		panic("Failed to create sub filesystem: " + err.Error())
	}

	// Asset handler
	assetHandler := http.StripPrefix("/assets/", http.FileServer(http.FS(fsys)))
	mux.Handle("/assets/", assetHandler)

	// SPA handler
	spaHandler := commonHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(ui.Assets, "dist/index.html")
		if err != nil {
			http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")

		if _, err := w.Write(data); err != nil {
			http.Error(w, "Failed to write index.html", http.StatusInternalServerError)
			return
		}
	}))
	mux.Handle("/", spaHandler)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r) // Replace with appropriate logic for handling requests
}

func Append(router *mux.Router, basePath string) error {
	return nil
}
