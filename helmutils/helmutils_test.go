package helmutils_test

import (
	"flag"
	"helm-api/defaults"
	"helm-api/helmutils"
	"helm-api/utils"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

// MockLogger is a mock implementation of the Logger interface using testify's mock.
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(args...)
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.Called(args...)
}

// mock_helm.go
type MockInstallAction struct {
	mock.Mock
}

func (m *MockInstallAction) Run(chart *chart.Chart, values map[string]interface{}) (*release.Release, error) {
	args := m.Called(chart, values)
	return args.Get(0).(*release.Release), args.Error(1)
}

type MockHelmActioner struct {
	mock.Mock
}

func (m *MockHelmActioner) NewInstall(config *action.Configuration) helmutils.InstallAction {
	args := m.Called(config)
	return args.Get(0).(helmutils.InstallAction)
}

func (m *MockHelmActioner) NewList(config *action.Configuration) helmutils.ListAction {
	args := m.Called(config)
	return args.Get(0).(helmutils.ListAction)
}

func (m *MockHelmActioner) NewUninstall(config *action.Configuration) helmutils.UninstallAction {
	args := m.Called(config)
	return args.Get(0).(helmutils.UninstallAction)
}

type MockListAction struct {
	mock.Mock
}

func (m *MockListAction) Run() ([]*release.Release, error) {
	args := m.Called()
	return args.Get(0).([]*release.Release), args.Error(1)
}

type MockChartLoader struct {
	mock.Mock
}

func (m *MockChartLoader) Load(path string) (*chart.Chart, error) {
	args := m.Called(path)
	return args.Get(0).(*chart.Chart), args.Error(1)
}

type MockUpgradeAction struct {
	mock.Mock
}

func (m *MockUpgradeAction) Run(name string, chart *chart.Chart, values map[string]interface{}) (*release.Release, error) {
	args := m.Called(name, chart, values)
	return args.Get(0).(*release.Release), args.Error(1)
}

// Add to MockHelmActioner
func (m *MockHelmActioner) NewUpgrade(config *action.Configuration) helmutils.UpgradeAction {
	args := m.Called(config)
	return args.Get(0).(helmutils.UpgradeAction)
}

type MockUninstallAction struct {
	mock.Mock
}

func (m *MockUninstallAction) Run(name string) (*release.UninstallReleaseResponse, error) {
	args := m.Called(name)
	return args.Get(0).(*release.UninstallReleaseResponse), args.Error(1)
}

type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) DeleteSubfolder(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) ReadValuesFile(path string) (*helmutils.Values, error) {
	args := m.Called(path)
	return args.Get(0).(*helmutils.Values), args.Error(1)
}

func (m *MockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	args := m.Called(path, data, perm)
	return args.Error(0)
}

func TestCreateHelmChartFromSource_Success(t *testing.T) {
	t.Parallel()
	// Initialize Helm settings
	settings := cli.New()

	// Initialize action configuration with memory driver (no actual Kubernetes cluster needed)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "default", "memory", func(format string, v ...interface{}) {
		// No-op logger for action.Configuration
	}); err != nil {
		t.Fatalf("Failed to initialize Helm action configuration: %v", err)
	}

	// Create a temporary source chart directory
	sourceDir, err := os.MkdirTemp("", "source-chart")
	if err != nil {
		t.Fatalf("Failed to create temporary source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir) // Clean up the temporary directory

	mockLogger := new(MockLogger)
	mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()

	// Create a minimal Chart.yaml in the source directory
	chartYamlContent := `apiVersion: v2
name: source-chart
description: A source Helm chart for testing
version: 0.1.0`

	t.Logf("chartYamlContent: %s\n", chartYamlContent)
	err = os.WriteFile(filepath.Join(sourceDir, "Chart.yaml"), []byte(chartYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	// Create a templates directory with a dummy template
	templatesDir := filepath.Join(sourceDir, "templates")
	err = os.MkdirAll(templatesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	deploymentYaml := `apiVersion: apps/v1
kind: Deployment
metadata:
	name: {{ .Chart.Name }}
spec:
	replicas: {{ .Values.replicaCount }}
	selector:
	matchLabels:
		app: {{ .Chart.Name }}
	template:
	metadata:
		labels:
		app: {{ .Chart.Name }}
	spec:
		containers:
		- name: {{ .Chart.Name }}
			image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
			ports:
			- containerPort: 80
		`
	err = os.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), []byte(deploymentYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to write deployment.yaml: %v", err)
	}

	// Create a temporary destination directory
	destDir, err := os.MkdirTemp("", "dest-chart")
	if err != nil {
		t.Fatalf("Failed to create temporary destination directory: %v", err)
	}
	defer os.RemoveAll(destDir) // Clean up

	// Create RealClient with mock logger
	helmClient := &helmutils.RealClient{
		ActionConfig: actionConfig,
		Logger:       mockLogger,
		Default: helmutils.Value{
			Namespace: "default",
			OutputDir: destDir,
			SourceDir: sourceDir,
		},
	}

	// Define chart options
	options := chart.Metadata{
		Name:        "chart1",
		Version:     "0.2.0",
		Description: "A new Helm chart created from source-chart",
	}

	// Call the function
	chartPathStr, err := helmClient.CreateHelmChartFromSource(options)
	if err != nil {
		t.Fatalf("CreateHelmChartFromSource: %v", err)
	}

	t.Logf("chartPathStr: %s\n", chartPathStr)

	// the actul Chart files will be stored in options.Name folder
	updatedDestDir := destDir + "/" + defaults.EnvPrefix + options.Name

	t.Logf("updatedDestDir: %s\n", updatedDestDir)

	// Verify that the new Chart.yaml exists and has correct metadata
	newChartYamlPath := filepath.Join(updatedDestDir, "Chart.yaml")
	if _, err := os.Stat(newChartYamlPath); os.IsNotExist(err) {
		t.Fatalf("Chart.yaml does not exist in destination directory")
	}

	loadedChart, err := loader.Load(updatedDestDir)
	if err != nil {
		t.Fatalf("Failed to load created chart: %v", err)
	}

	if loadedChart.Name() != defaults.EnvPrefix+options.Name {
		t.Errorf("Expected chart name '%s', got '%s'", defaults.EnvPrefix+options.Name, loadedChart.Name())
	}

	if loadedChart.Metadata.Version != options.Version {
		t.Errorf("Expected chart version '%s', got '%s'", options.Version, loadedChart.Metadata.Version)
	}

	if loadedChart.Metadata.Description != options.Description {
		t.Errorf("Expected chart description '%s', got '%s'", options.Description, loadedChart.Metadata.Description)
	}

	// Verify that templates are copied
	expectedTemplates := []string{"deployment.yaml"}
	for _, tmpl := range expectedTemplates {
		tmplPath := filepath.Join(updatedDestDir, "templates", tmpl)
		if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
			t.Errorf("Expected template '%s' does not exist in the new chart", tmpl)
		}
	}
}

func TestCreateHelmChartFromSource_InvalidSource(t *testing.T) {
	t.Parallel()
	// Initialize Helm settings
	settings := cli.New()

	// Initialize action configuration with memory driver (no actual Kubernetes cluster needed)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "default", "memory", func(format string, v ...interface{}) {
		// No-op logger for action.Configuration
	}); err != nil {
		t.Fatalf("Failed to initialize Helm action configuration: %v", err)
	}

	mockLogger := new(MockLogger)
	mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()

	// Create a temporary destination directory
	destDir, err := os.MkdirTemp("", "dest-chart")
	if err != nil {
		t.Fatalf("Failed to create temporary destination directory: %v", err)
	}
	defer os.RemoveAll(destDir) // Clean up

	// Call the function with invalid source path
	invalidSource := "/invalid/source/path"

	// Create RealClient with mock logger
	helmClient := &helmutils.RealClient{
		ActionConfig: actionConfig,
		Logger:       mockLogger,
		Default: helmutils.Value{
			Namespace: "default",
			OutputDir: destDir,
			SourceDir: invalidSource,
		},
	}

	// Define chart options
	options := chart.Metadata{
		Name:        "newchart",
		Version:     "0.2.0",
		Description: "A new Helm chart created from source-chart",
	}

	_, err = helmClient.CreateHelmChartFromSource(options)
	if err == nil {
		t.Fatalf("Expected error when source path is invalid, but got none")
	}

	expectedErrorMsg := "source chart path does not exist: /invalid/source/path"
	if !utils.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestCreateHelmChartFromSource_InvalidDestination(t *testing.T) {
	t.Parallel()
	// Initialize Helm settings
	settings := cli.New()

	// Initialize action configuration with memory driver (no actual Kubernetes cluster needed)
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "default", "memory", func(format string, v ...interface{}) {
		// No-op logger for action.Configuration
	}); err != nil {
		t.Fatalf("Failed to initialize Helm action configuration: %v", err)
	}

	mockLogger := new(MockLogger)
	mockLogger.On("Debug", mock.Anything, mock.Anything, mock.Anything).Return()

	// Create a temporary source chart directory
	sourceDir, err := os.MkdirTemp("", "source-chart")
	if err != nil {
		t.Fatalf("Failed to create temporary source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir) // Clean up

	invalidDest := "/invalid/dest/path"

	helmClient := &helmutils.RealClient{
		ActionConfig: actionConfig,
		Logger:       mockLogger,
		Default: helmutils.Value{
			Namespace: "default",
			OutputDir: invalidDest,
			SourceDir: sourceDir,
		},
	}
	// Create a minimal Chart.yaml in the source directory
	chartYamlContent := `apiVersion: v2
name: source-chart
description: A source Helm chart for testing
version: 0.1.0
`
	err = os.WriteFile(filepath.Join(sourceDir, "Chart.yaml"), []byte(chartYamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write Chart.yaml: %v", err)
	}

	// Define chart options
	options := chart.Metadata{
		Name:        "newchart",
		Version:     "0.2.0",
		Description: "A new Helm chart created from source-chart",
	}

	_, err = helmClient.CreateHelmChartFromSource(options)
	if err == nil {
		t.Fatalf("Expected error when destination path is invalid, but got none")
	}

	expectedErrorMsg := "output directory does not exist: /invalid/dest/path"
	if !utils.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestListReleases(t *testing.T) {
	// Setup
	mockActioner := &MockHelmActioner{}
	mockLogger := new(MockLogger)
	mockList := &MockListAction{}

	client := &helmutils.RealClient{
		ActionConfig: new(action.Configuration),
		Logger:       mockLogger,
		Actioner:     mockActioner,
	}

	// Setup expectations
	mockLogger.On("Infof", mock.Anything, mock.Anything).Return()
	mockActioner.On("NewList", mock.Anything).Return(mockList)
	mockList.On("Run").Return([]*release.Release{
		{Name: "test-release-1"},
		{Name: "test-release-2"},
	}, nil)

	// Test
	releases, err := client.ListReleases()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []string{"test-release-1", "test-release-2"}, releases)
	mockActioner.AssertExpectations(t)
	mockList.AssertExpectations(t)
}

func TestInstallRelease(t *testing.T) {
	// Setup
	mockInstall := &MockInstallAction{}
	mockActioner := &MockHelmActioner{}
	mockLogger := new(MockLogger)
	mockList := &MockListAction{}
	mockChart := &chart.Chart{}
	mockChartLoader := new(MockChartLoader) // Create instance

	actionConfig := new(action.Configuration)

	client := &helmutils.RealClient{
		ActionConfig: actionConfig,
		Logger:       mockLogger,
		Actioner:     mockActioner,
		ChartLoader:  mockChartLoader,
	}

	// Setup mock expectations
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return()

	// Mock list releases
	mockActioner.On("NewList", actionConfig).Return(mockList)
	mockList.On("Run").Return([]*release.Release{}, nil)

	mockChartLoader.On("Load", "test-chart").Return(mockChart, nil)

	// Mock install
	mockActioner.On("NewInstall", actionConfig).Return(mockInstall)
	mockInstall.On("Run", mockChart, mock.Anything).Return(&release.Release{}, nil)

	// Test
	_, err := client.InstallRelease("test-chart", "test-release")

	// Assertions
	assert.NoError(t, err)
	mockActioner.AssertExpectations(t)
	mockInstall.AssertExpectations(t)
	mockList.AssertExpectations(t)
	mockChartLoader.AssertExpectations(t)

}

func TestUpgradeRelease(t *testing.T) {
	// Setup
	mockActioner := &MockHelmActioner{}
	mockLogger := new(MockLogger)
	mockList := &MockListAction{}
	mockUpgrade := &MockUpgradeAction{}
	mockChartLoader := new(MockChartLoader)
	mockChart := &chart.Chart{}

	client := &helmutils.RealClient{
		ActionConfig: new(action.Configuration),
		Logger:       mockLogger,
		Actioner:     mockActioner,
		ChartLoader:  mockChartLoader,
	}

	releaseName := "test-release"
	chartPath := filepath.Join(client.Default.OutputDir, releaseName)

	// Mock expectations
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return()

	mockActioner.On("NewList", mock.Anything).Return(mockList)
	mockList.On("Run").Return([]*release.Release{
		{Name: releaseName},
	}, nil)

	mockChartLoader.On("Load", chartPath).Return(mockChart, nil)

	mockActioner.On("NewUpgrade", mock.Anything).Return(mockUpgrade)
	mockUpgrade.On("Run", releaseName, mockChart, mock.Anything).Return(&release.Release{}, nil)

	// Test
	_, err := client.UpgradeRelease(releaseName)

	// Assert
	assert.NoError(t, err)
	mockActioner.AssertExpectations(t)
	mockUpgrade.AssertExpectations(t)
	mockList.AssertExpectations(t)
	mockChartLoader.AssertExpectations(t)
}

func TestUninstallRelease(t *testing.T) {
	// Setup
	mockActioner := &MockHelmActioner{}
	mockLogger := new(MockLogger)
	mockList := &MockListAction{}
	mockUninstall := &MockUninstallAction{}
	mockFS := new(MockFileSystem)

	client := &helmutils.RealClient{
		ActionConfig: new(action.Configuration),
		Logger:       mockLogger,
		Actioner:     mockActioner,
		Filesystem:   mockFS,
	}

	releaseName := "test-release"
	chartPath := filepath.Join(client.Default.OutputDir, releaseName)

	// Mock expectations
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return()

	mockActioner.On("NewList", mock.Anything).Return(mockList)
	mockList.On("Run").Return([]*release.Release{
		{Name: releaseName},
	}, nil)

	mockActioner.On("NewUninstall", mock.Anything).Return(mockUninstall)
	mockUninstall.On("Run", releaseName).Return(&release.UninstallReleaseResponse{
		Release: &release.Release{},
	}, nil)

	mockFS.On("DeleteSubfolder", chartPath).Return(nil)

	// Test
	_, err := client.UninstallRelease(releaseName)

	// Assert
	assert.NoError(t, err)
	mockActioner.AssertExpectations(t)
	mockUninstall.AssertExpectations(t)
	mockList.AssertExpectations(t)
	mockFS.AssertExpectations(t)
}

func TestUpdateValuesFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Setup test structure
	releaseName := "test-release"
	replicaCount := 2

	// Create the release directory
	releasePath := filepath.Join(tmpDir, releaseName)
	err := os.MkdirAll(releasePath, 0755)
	assert.NoError(t, err)

	// Create initial values.yaml
	valuesFile := filepath.Join(releasePath, "values.yaml")
	initialValues := map[string]interface{}{
		"replicas": 1,
		"other":    "value",
	}

	valueBytes, err := yaml.Marshal(initialValues)
	assert.NoError(t, err)

	err = os.WriteFile(valuesFile, valueBytes, 0644)
	assert.NoError(t, err)

	// Create client with temp directory
	client := &helmutils.RealClient{
		Default: helmutils.Value{ // Use the correct struct type
			Namespace: "default",
			OutputDir: tmpDir,
		},
	}

	// Run the update
	err = client.UpdateValuesFile(releaseName, replicaCount)
	assert.NoError(t, err)

	// Verify the changes
	updatedBytes, err := os.ReadFile(valuesFile)
	assert.NoError(t, err)

	var updatedValues map[string]interface{}
	err = yaml.Unmarshal(updatedBytes, &updatedValues)
	assert.NoError(t, err)

	assert.Equal(t, replicaCount, updatedValues["replicas"])
	assert.Equal(t, "value", updatedValues["other"]) // Verify other values remain
}

func TestNewRealClient(t *testing.T) {
	// Save original env vars to restore later

	originalEnv := map[string]string{
		"HELM_API_NAMESPACE":       os.Getenv("HELM_API_NAMESPACE"),
		"HELM_API_HELM_OUT_DIR":    os.Getenv("HELM_API_HELM_OUT_DIR"),
		"HELM_API_HELM_SOURCE_DIR": os.Getenv("HELM_API_HELM_SOURCE_DIR"),
		"HELM_DRIVER":              os.Getenv("HELM_DRIVER"),
		"HELM_API_CREATE_API_KEY":  os.Getenv("HELM_API_CREATE_API_KEY"),
		"HELM_API_DELETE_API_KEY":  os.Getenv("HELM_API_DELETE_API_KEY"),
		"HELM_API_UPDATE_API_KEY":  os.Getenv("HELM_API_UPDATE_API_KEY"),
	}

	// Cleanup function to restore environment
	defer func() {
		for k, v := range originalEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	tests := []struct {
		name      string
		envVars   map[string]string
		logger    *MockLogger
		wantError bool
	}{
		{
			name: "successful initialization with env vars",
			envVars: map[string]string{
				"HELM_API_NAMESPACE":       "test-ns",
				"HELM_API_HELM_OUT_DIR":    "/test/out",
				"HELM_API_HELM_SOURCE_DIR": "/test/source",
				"HELM_DRIVER":              "secrets",
				"HELM_API_CREATE_API_KEY":  "create-key",
				"HELM_API_DELETE_API_KEY":  "delete-key",
				"HELM_API_UPDATE_API_KEY":  "update-key",
			},
			logger:    &MockLogger{},
			wantError: false,
		},
		{
			name: "missing create API key",
			envVars: map[string]string{
				"HELM_API_DELETE_API_KEY": "delete-key",
				"HELM_API_UPDATE_API_KEY": "update-key",
			},
			logger:    &MockLogger{},
			wantError: true,
		},
		{
			name: "nil logger should use default",
			envVars: map[string]string{
				"HELM_API_CREATE_API_KEY": "create-key",
				"HELM_API_DELETE_API_KEY": "delete-key",
				"HELM_API_UPDATE_API_KEY": "update-key",
			},
			logger:    &MockLogger{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Clear and set environment variables
			for k := range originalEnv {
				os.Unsetenv(k)
			}
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Test the constructor
			client, err := helmutils.NewRealClient(tt.logger)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, client)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, client)

			// Verify client configuration
			if val, exists := tt.envVars["HELM_API_NAMESPACE"]; exists {
				assert.Equal(t, val, client.Default.Namespace)
			}
			if val, exists := tt.envVars["HELM_API_HELM_OUT_DIR"]; exists {
				assert.Equal(t, val, client.Default.OutputDir)
			}
			if val, exists := tt.envVars["HELM_API_HELM_SOURCE_DIR"]; exists {
				assert.Equal(t, val, client.Default.SourceDir)
			}
		})
	}
}
