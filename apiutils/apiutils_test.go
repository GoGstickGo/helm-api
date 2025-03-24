package apiutils_test

import (
	"helm-api/apiutils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestValidateEndpoint(t *testing.T) {
	// Save original env vars to restore later
	originalEnvVars := map[string]string{
		"HELM_API_CREATE_API_KEY": os.Getenv("HELM_API_CREATE_API_KEY"),
		"HELM_API_UPDATE_API_KEY": os.Getenv("HELM_API_UPDATE_API_KEY"),
		"HELM_API_DELETE_API_KEY": os.Getenv("HELM_API_DELETE_API_KEY"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalEnvVars {
			os.Setenv(k, v)
		}
	}()

	// Set test env vars
	os.Setenv("HELM_API_CREATE_API_KEY", "create-key")
	os.Setenv("HELM_API_UPDATE_API_KEY", "update-key")
	os.Setenv("HELM_API_DELETE_API_KEY", "delete-key")

	tests := []struct {
		name     string
		path     string
		apiKey   string
		expected bool
	}{
		{"valid create endpoint", "/api/v1/create-env", "create-key", true},
		{"invalid create key", "/api/v1/create-env", "wrong-key", false},
		{"valid update endpoint", "/api/v1/update-env", "update-key", true},
		{"invalid update key", "/api/v1/update-env", "wrong-key", false},
		{"valid delete endpoint", "/api/v1/delete-env", "delete-key", true},
		{"invalid delete key", "/api/v1/delete-env", "wrong-key", false},
		{"health check no auth", "/api/v1/health-check", "", true},
		{"health check with key", "/api/v1/health-check", "any-key", true},
		{"list no auth", "/api/v1/list", "", true},
		{"list with key", "/api/v1/list", "any-key", true},
		{"unknown endpoint", "/api/v1/unknown", "any-key", false},
		{"empty path", "", "any-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := apiutils.ValidateEndpoint(tt.path, tt.apiKey)
			if result != tt.expected {
				t.Errorf("ValidateEndpoint(%q, %q) = %v, want %v",
					tt.path, tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	// Save original env vars
	originalEnvVars := map[string]string{
		"HELM_API_CREATE_API_KEY": os.Getenv("HELM_API_CREATE_API_KEY"),
		"HELM_API_UPDATE_API_KEY": os.Getenv("HELM_API_UPDATE_API_KEY"),
		"HELM_API_DELETE_API_KEY": os.Getenv("HELM_API_DELETE_API_KEY"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalEnvVars {
			os.Setenv(k, v)
		}
	}()

	// Set test env vars
	os.Setenv("HELM_API_CREATE_API_KEY", "create-key")
	os.Setenv("HELM_API_UPDATE_API_KEY", "update-key")
	os.Setenv("HELM_API_DELETE_API_KEY", "delete-key")

	tests := []struct {
		name           string
		path           string
		apiKey         string
		expectedStatus int
		handlerCalled  bool
	}{
		{"valid create endpoint", "/api/v1/create-env", "create-key", http.StatusOK, true},
		{"invalid create key", "/api/v1/create-env", "wrong-key", http.StatusUnauthorized, false},
		{"health check no auth", "/api/v1/health-check", "", http.StatusOK, true},
		{"unknown endpoint", "/api/v1/unknown", "any-key", http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock handler that will be wrapped by the middleware
			var handlerCalled bool
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			// Create the middleware
			middleware := apiutils.AuthMiddleware(mockHandler)

			// Create a test request
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Add API key if provided
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			middleware.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}

			// Check if handler was called
			if handlerCalled != tt.handlerCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.handlerCalled)
			}
		})
	}
}
