// pkg/server/service/managerfactory.go
package service

import (
	"net/http"

	"github.com/containous/mux"
	"github.com/pentora-ai/pentora/pkg/api"
	"github.com/pentora-ai/pentora/pkg/api/dashboard"
	"github.com/pentora-ai/pentora/pkg/config/runtime"
	"github.com/pentora-ai/pentora/pkg/config/static"
	"github.com/rs/zerolog/log"
)

// ManagerFactory a factory of service manager.
type ManagerFactory struct {
	api              func(configuration *runtime.Configuration) http.Handler
	dashboardHandler http.Handler
}

// NewManagerFactory creates a new ManagerFactory.
func NewManagerFactory(staticConfiguration static.Configuration) *ManagerFactory {
	factory := &ManagerFactory{}

	if staticConfiguration.API != nil {
		apiRouterBuilder := api.NewBuilder(staticConfiguration)

		if staticConfiguration.API.Dashboard {
			factory.dashboardHandler = dashboard.Handler{BasePath: staticConfiguration.API.BasePath}
			factory.api = func(configuration *runtime.Configuration) http.Handler {
				router := apiRouterBuilder(configuration).(*mux.Router)
				if err := dashboard.Append(router, staticConfiguration.API.BasePath); err != nil {
					log.Error().Err(err).Msg("Error appending dashboard to API router")
				}
				return router
			}
		} else {
			factory.api = apiRouterBuilder
		}
	}

	return factory
}
