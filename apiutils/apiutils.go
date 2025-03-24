package apiutils

import (
	"net/http"
	"os"
	"strings"
)

type EndpointConfig struct {
	Path   string
	APIKey string
	NoAuth bool
}

func ValidateEndpoint(path, apiKey string) bool {
	endpoints := []EndpointConfig{
		{
			Path:   "create-env",
			APIKey: os.Getenv("HELM_API_CREATE_API_KEY"),
		},
		{
			Path:   "update-env",
			APIKey: os.Getenv("HELM_API_UPDATE_API_KEY"),
		},
		{
			Path:   "delete-env",
			APIKey: os.Getenv("HELM_API_DELETE_API_KEY"),
		},
		{
			Path:   "health-check",
			NoAuth: true,
		},
		{
			Path:   "list",
			NoAuth: true,
		},
	}

	for _, endpoint := range endpoints {
		if strings.Contains(path, endpoint.Path) {
			if endpoint.NoAuth {

				return true
			}

			return endpoint.APIKey == apiKey
		}
	}

	return false
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")

		if ValidateEndpoint(r.URL.Path, apiKey) {
			next.ServeHTTP(w, r)

			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
