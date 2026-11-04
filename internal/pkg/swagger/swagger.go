package swagger

import (
	"log/slog"
	"net/http"

	"bnn/docs"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const specPath = "/swagger/openapi.yaml"

func Register(router *mux.Router) {
	router.HandleFunc(specPath, spec).Methods(http.MethodGet)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL(specPath),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	)).Methods(http.MethodGet)
}

func spec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	if _, err := w.Write(docs.OpenAPI); err != nil {
		slog.Error("Failed to write spec", "error", err)
	}
}
