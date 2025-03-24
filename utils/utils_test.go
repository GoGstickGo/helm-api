package utils_test

import (
	"helm-api/utils"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetEnvOrDefault(t *testing.T) {
	t.Parallel()
	type args struct {
		key          string
		defaultValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "successful #1",
			args: args{key: "HELM_API_NAMESPACE", defaultValue: "helm-api-pg"},
			want: "helm-api-pg",
		},
		{
			name: "successful #2",
			args: args{key: "HELM_API_NAMESPACE", defaultValue: "test-helm-api"},
			want: "test-helm-api",
		},
		{
			name: "failed #3",
			args: args{key: "HELM_API_NAMESPACE", defaultValue: ""},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Mark each sub-test as parallel.

			got := utils.GetEnvOrDefault(tt.args.key, tt.args.defaultValue)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("test failed, diff ==> %v\n,", diff)
			}
		})
	}
}

func TestValidateAPIKeys(t *testing.T) {
	t.Parallel()

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

	type args struct {
		required []string
	}

	tests := []struct {
		name     string
		setupEnv func()
		args     args
		wantErr  bool
	}{
		// TODO: Add test cases.
		{
			name:    "successful #1",
			args:    args{required: []string{"HELM_API_UPDATE_API_KEY", "HELM_API_DELETE_API_KEY", "HELM_API_CREATE_API_KEY"}},
			wantErr: false,
		},
		{
			name:    "failed #1",
			args:    args{required: []string{"HELM_API_UPDATE_API_KEY", "HELM_API_DELETE_API_KEY"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.ValidateAPIKeys(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
