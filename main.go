package main

import (
	"context"
	"encoding/json"
	"fmt"
	"helm-api/apiutils"
	"helm-api/awsutils"
	"helm-api/defaults"
	"helm-api/helmutils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/chart"
)

// Response represents a standard API response.
type Response struct {
	Message string   `json:"message"`
	Error   string   `json:"error,omitempty"`
	Data    []string `json:"data,omitempty"`
}

type ScaleAction string

const (
	ScaleUp   ScaleAction = "up"
	ScaleDown ScaleAction = "down"
)

type Request struct {
	ChartMetadata chart.Metadata `json:"chartMetadata"`
	Action        *ScaleAction   `json:"action,omitempty"`
}

func main() {
	// Initialize a custom logger (e.g., JSONFormatter)
	customLogger := logrus.New()
	customLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	customLogger.SetLevel(logrus.InfoLevel)

	if os.Getenv("HELM_API_AWS") == "true" {

		// Create a root context with a timeout for the entire application run.
		ctxTimeOut, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Initialize AWS clients.
		configLoader := &awsutils.RealAWSConfigLoader{}
		clients, err := awsutils.InitializeAWSClients(ctxTimeOut, configLoader, defaults.AwsRegion)
		if err != nil {
			customLogger.Fatalf("AWS auth error: %v", err)
		}
		customLogger.Info("AWS client initialized successfully")

		// Setting API keys
		if err = awsutils.GetSSMParameters(ctxTimeOut, clients.SSM, defaults.SsmParams); err != nil {
			customLogger.Fatalf("Setting master API keys failed: %v", err)
		}

	}

	// Initialize the helm client
	helmClient, err := helmutils.NewRealClient(customLogger)
	if err != nil {
		customLogger.Fatalf("Failed to create Helm client: %v", err)
	}

	customLogger.Info("Helm client initialized successfully")

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	//Validate API Key
	r.Use(apiutils.AuthMiddleware)

	// Routes
	r.Post("/create-env", createEnvHandler(helmClient))
	r.Post("/update-env/{chartName}", updateEnvHandler(helmClient))
	r.Post("/delete-env/{chartName}", deleteEnvHandler(helmClient))
	r.Get("/health-check", healthCheck)
	r.Get("/list", listEnvHandler(helmClient))

	// Create server
	port := os.Getenv("HELM_API_PORT")
	if port == "" {
		port = defaults.Port
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the server
	go func() {
		customLogger.Printf("Server is starting on port %s...", port)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking select waiting for either server errors or context cancellation.
	select {
	case err := <-serverErrors:
		customLogger.Fatalf("Error starting server: %v", err)

	case sig := <-shutdown:
		customLogger.Printf("Start shutdown... Signal: %v", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Shut down gracefully, but wait no longer than the context timeout.
		err := server.Shutdown(ctx)
		if err != nil {
			customLogger.Printf("Graceful shutdown did not complete in %v : %v", 15*time.Second, err)
			err = server.Close()
			if err != nil {
				customLogger.Printf("Error killing server : %v", err)
			}
		}

		// Exit with status code 1 if there was an error shutting down
		if err != nil {
			os.Exit(1)
		}
	}

}

// createEnvHandler handles the creation of Helm chart resources.
func createEnvHandler(hc *helmutils.RealClient) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp := Response{
				Message: "Invalid request payload",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		// Validate the input.
		if req.ChartMetadata.Name == "" {
			http.Error(w, "Missing required fields in request", http.StatusBadRequest)

			return
		}

		// Call CreateHelmChartFromSource.
		chartPath, err := hc.CreateHelmChartFromSource(req.ChartMetadata)
		if err != nil {
			resp := Response{
				Message: "Failed to create Helm chart",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		_, err = hc.InstallRelease(chartPath, req.ChartMetadata.Name)
		if err != nil {
			resp := Response{
				Message: "Failed to install Helm chart",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		// Success response.
		resp := Response{
			Message: fmt.Sprintf("Helm chart %s created and successfully installed %s", req.ChartMetadata.Name, hc.Default.OutputDir),
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)

			return
		}
	}
}

// createEnvHandler handles the creation of Helm chart resources.
func updateEnvHandler(hc *helmutils.RealClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chartName := chi.URLParam(r, "chartName")

		// Validate the input
		if chartName == "" {
			http.Error(w, "Missing required fields in request", http.StatusBadRequest)

			return
		}

		releaseName := defaults.EnvPrefix + chartName

		// Check if helm-chart exists
		if _, err := os.Stat(hc.Default.OutputDir + "/" + releaseName); os.IsNotExist(err) {
			http.Error(w, "env doesn't exists,please use creata-env endpoint for brand new env", http.StatusBadRequest)

			return
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp := Response{
				Message: "Invalid request payload",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		if req.Action != nil {
			replicas := map[ScaleAction]int{
				ScaleUp:   1,
				ScaleDown: 0,
			}

			if count, exists := replicas[*req.Action]; exists {
				if err := hc.UpdateValuesFile(releaseName, count); err != nil {
					resp := Response{
						Message: "Updating values.yaml failed",
						Error:   err.Error(),
					}
					w.WriteHeader(http.StatusInternalServerError)
					if err := json.NewEncoder(w).Encode(resp); err != nil {
						http.Error(w, "Failed to encode response", http.StatusInternalServerError)

						return
					}

					return
				}

			}

			_, err := hc.UpgradeRelease(releaseName)
			if err != nil {
				resp := Response{
					Message: "Failed to update Helm chart",
					Error:   err.Error(),
				}
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					http.Error(w, "Failed to encode response", http.StatusInternalServerError)

					return
				}

				return
			}

			// Success response.
			resp := Response{
				Message: fmt.Sprintf("Helm chart %s successfully updated %s", req.ChartMetadata.Name, hc.Default.OutputDir),
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "API is healthy",
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusServiceUnavailable)

		return
	}
}

func deleteEnvHandler(hc *helmutils.RealClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chartName := chi.URLParam(r, "chartName")

		// Validate the input.
		if chartName == "" {
			http.Error(w, "Missing required fields in request", http.StatusBadRequest)

			return
		}

		releaseName := defaults.EnvPrefix + chartName
		_, err := hc.UninstallRelease(releaseName)
		if err != nil {
			resp := Response{
				Message: "Failed to uninstall Helm chart",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		// Success response.
		resp := Response{
			Message: fmt.Sprintf("Helm chart %s successfully uninstalled", releaseName),
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)

			return
		}

	}
}

func listEnvHandler(hc *helmutils.RealClient) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		chartList, err := hc.ListReleases()
		if err != nil {
			resp := Response{
				Message: "Failed to list helm chart with prefix helm-api-",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)

				return
			}

			return
		}

		message := "No helm-api related helm chart"
		if len(chartList) > 0 {
			message = "List:"
		}

		resp := Response{
			Message: message,
			Data:    chartList,
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)

			return
		}

	}
}
