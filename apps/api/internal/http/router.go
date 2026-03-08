package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		registerDashboardRoutes(r)
		registerGatewayRoutes(r)
		registerSessionRoutes(r)
		registerApprovalRoutes(r)
		registerPolicyRoutes(r)
		registerAlertRoutes(r)
		registerInventoryRoutes(r)
	})

	registerFrontendRoutes(r)

	return r
}
