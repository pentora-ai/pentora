// pkg/api/handler.go
package api

import (
	"net/http"

	"github.com/containous/mux"
	"github.com/pentora-ai/pentora/pkg/config/runtime"
	"github.com/pentora-ai/pentora/pkg/config/static"
)

type apiError struct {
	Message string `json:"message"`
}

// Handler serves the configuration and status of Pentora on API endpoints.
type Handler struct {
	staticConfiguration static.Configuration

	// runtimeConfiguration is the data set used to create all the data representations exposed by the API.
	runtimeConfiguration *runtime.Configuration
}

// NewBuilder returns a http.Handler builder
func NewBuilder(staticConfiguration static.Configuration) func(*runtime.Configuration) http.Handler {
	return func(configuration *runtime.Configuration) http.Handler {
		return New(staticConfiguration, configuration).createRouter()
	}
}

// New returns a Handler defined by staticConfig, and if provided, by runtimeConfig.
// It finishes populating the information provided in the runtimeConfig.
func New(staticConfiguration static.Configuration, runtimeConfiguration *runtime.Configuration) *Handler {
	rConfig := runtimeConfiguration
	if rConfig == nil {
		rConfig = &runtime.Configuration{}
	}

	return &Handler{

		staticConfiguration: staticConfiguration,
	}
}

// createRouter creates API routes and router.
func (h Handler) createRouter() *mux.Router {
	router := mux.NewRouter().UseEncodedPath()
	// todo add routes
	return router
}
