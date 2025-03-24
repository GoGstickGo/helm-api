// helm_client.go
package helmutils

import (
	"flag"
	"fmt"
	"helm-api/defaults"
	"helm-api/utils"
	"os"
	"path/filepath"
	"slices"
	"time"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

// NewRealHelmClient initializes and returns a RealClient.
// Load configuration from flags and environment
func loadConfig() (*Config, error) {
	// Parse flags
	namespace := flag.String("namespace", defaults.NameSpace, "Namespace where the environment should be created")
	outputDir := flag.String("outputDir", defaults.OutPutDir, "Folder where charts helm chart will be stored")
	sourceDir := flag.String("sourceDir", defaults.SourceDir, "Folder where default helm chart is stored")
	helmDriver := flag.String("helmDriver", defaults.HelmDriver, "HELM driver")

	// Override with environment variables
	config := &Config{
		Namespace:  utils.GetEnvOrValue("HELM_API_NAMESPACE", *namespace),
		OutputDir:  utils.GetEnvOrValue("HELM_API_HELM_OUT_DIR", *outputDir),
		SourceDir:  utils.GetEnvOrValue("HELM_API_HELM_SOURCE_DIR", *sourceDir),
		HelmDriver: utils.GetEnvOrValue("HELM_DRIVER", *helmDriver),
	}

	return config, nil
}

// Main constructor
func NewRealClient(logger Logger) (*RealClient, error) {
	// Setup logger
	logger = setupLogger(logger)

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate API keys
	if err := utils.ValidateAPIKeys(); err != nil {
		return nil, err
	}

	// Initialize Helm configuration
	settings := cli.New()
	actionConfig := new(action.Configuration)
	err = actionConfig.Init(
		settings.RESTClientGetter(),
		config.Namespace,
		config.HelmDriver,
		func(format string, v ...interface{}) {
			logger.Infof(format, v...)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Helm action configuration: %w", err)
	}

	return &RealClient{
		ActionConfig: actionConfig,
		Logger:       logger,
		Default: Value{
			Namespace: config.Namespace,
			OutputDir: config.OutputDir,
			SourceDir: config.SourceDir,
		},
	}, nil
}

func (hc *RealClient) CreateHelmChartFromSource(options chart.Metadata) (chartPath string, err error) {
	// Validate source path existence.
	hc.Logger.Debug("CreateHelmChartFromSource:sourceDir:%s\n", hc.Default.SourceDir)
	if _, err := os.Stat(hc.Default.SourceDir); os.IsNotExist(err) {

		return "", fmt.Errorf("source chart path does not exist: %s", hc.Default.SourceDir)
	}

	// Validate output directory existence.
	if _, err := os.Stat(hc.Default.OutputDir); os.IsNotExist(err) {

		return "", fmt.Errorf("output directory does not exist: %s", hc.Default.OutputDir)
	}

	options.Name = defaults.EnvPrefix + options.Name

	chartPath = hc.Default.OutputDir + "/" + options.Name

	if _, err := os.Stat(chartPath); err == nil {
		hc.Logger.Infof("Helm chart already exist for %v ", options.Name)
		hc.Logger.Info("Skipping helm chart creation.")

		return chartPath, nil
	}

	// Create the new chart from the source chart.
	hc.Logger.Infof("Creating new Helm chart '%s' from source '%s' into destination '%s'", options.Name, hc.Default.SourceDir, hc.Default.OutputDir)
	if err = chartutil.CreateFrom(&options, hc.Default.OutputDir, hc.Default.SourceDir); err != nil {

		return "", fmt.Errorf("failed to create chart from source: %w", err)
	}

	hc.Logger.Infof("Successfully created Helm chart '%s' at '%s'", options.Name, hc.Default.OutputDir)

	return chartPath, nil
}

func (hc *RealClient) InstallRelease(chartPath, releaseName string) (*release.Release, error) {

	// Get all helm-api related helm releases.
	chartList, err := hc.ListReleases()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	// Make sure don't try to install already existing  helm-api release.
	if slices.Contains(chartList, releaseName) {
		return nil, fmt.Errorf("release for %s already exist please use update-env endpoint", releaseName)
	}

	var values map[string]interface{}

	releaseName = defaults.EnvPrefix + releaseName

	installClient := hc.Actioner.NewInstall(hc.ActionConfig)

	// Type assert to set specific fields
	if ic, ok := installClient.(*action.Install); ok {
		ic.ReleaseName = releaseName
		ic.Namespace = hc.Default.Namespace
		ic.Wait = true
		ic.Timeout = 300 * time.Second
		ic.DryRun = false
	}

	chart, err := hc.ChartLoader.Load(chartPath)
	if err != nil {

		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	rel, err := installClient.Run(chart, values)
	if err != nil {

		return nil, fmt.Errorf("%w", err)
	}

	// Log the installed manifests
	hc.Logger.Debug("Installed Release Manifests:")
	hc.Logger.Debug("----------------------------")
	hc.Logger.Debug(rel.Manifest)
	hc.Logger.Debug("----------------------------")

	return rel, nil
}

func (hc *RealClient) UpgradeRelease(releaseName string) (*release.Release, error) {
	// Get all helm-api related helm releases
	chartList, err := hc.ListReleases()
	if err != nil {

		return nil, fmt.Errorf("%w", err)
	}

	if !slices.Contains(chartList, releaseName) {

		return nil, fmt.Errorf("release name doesn't match any of helm-api related environments, please use correct release name: %s", releaseName)
	}

	chartPath := filepath.Join(hc.Default.OutputDir, releaseName)
	var values map[string]interface{}

	upgradeClient := hc.Actioner.NewUpgrade(hc.ActionConfig)

	// Type assert to set specific fields
	if uc, ok := upgradeClient.(*action.Upgrade); ok {
		uc.Namespace = hc.Default.Namespace
		uc.Wait = true
		uc.Timeout = 300 * time.Second
		uc.DryRun = false
		uc.Install = true
		uc.Force = true
	}

	chart, err := hc.ChartLoader.Load(chartPath)
	if err != nil {

		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	rel, err := upgradeClient.Run(releaseName, chart, values)
	if err != nil {

		return nil, fmt.Errorf("%w", err)
	}

	hc.Logger.Debug("Updated Release Manifests:")
	hc.Logger.Debug("----------------------------")
	hc.Logger.Debug(rel.Manifest)
	hc.Logger.Debug("----------------------------")

	return rel, nil
}

func (hc *RealClient) UninstallRelease(releaseName string) (*release.UninstallReleaseResponse, error) {
	chartList, err := hc.ListReleases()
	if err != nil {

		return nil, fmt.Errorf("%w", err)
	}

	if !slices.Contains(chartList, releaseName) {

		return nil, fmt.Errorf("release name doesn't match any of helm-api related environments, please use correct release name: %s", releaseName)
	}

	uninstallClient := hc.Actioner.NewUninstall(hc.ActionConfig)

	// Type assert to set specific fields
	if uc, ok := uninstallClient.(*action.Uninstall); ok {
		uc.DryRun = false
		uc.Timeout = 300 * time.Second
		uc.Wait = true
		uc.IgnoreNotFound = false
	}

	hc.Logger.Infof("Uninstall helm chart for '%s'", releaseName)
	rel, err := uninstallClient.Run(releaseName)
	if err != nil {

		return nil, fmt.Errorf("%w", err)
	}

	hc.Logger.Debug("Removed release info:")
	hc.Logger.Debug("----------------------------")
	hc.Logger.Debug(rel.Release.Info)
	hc.Logger.Debug("----------------------------")

	hc.Logger.Infof("Chart files removed from the storage")
	chartPath := filepath.Join(hc.Default.OutputDir, releaseName)
	if err := hc.Filesystem.DeleteSubfolder(chartPath); err != nil {
		return nil, fmt.Errorf("failed to delete chart files: %w", err)
	}

	return rel, nil
}

func (hc *RealClient) ListReleases() (chartList []string, err error) {
	listClient := hc.Actioner.NewList(hc.ActionConfig)

	// Type assert to set specific fields if needed
	if lc, ok := listClient.(*action.List); ok {
		lc.AllNamespaces = false
		lc.Filter = defaults.EnvPrefix
		lc.SetStateMask()
	}

	hc.Logger.Infof("List helm chart for namespace: '%s'", hc.Default.Namespace)

	rel, err := listClient.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run list: %w", err)
	}

	for _, v := range rel {
		chartList = append(chartList, v.Name)
	}

	return chartList, nil
}

func (hc *RealClient) UpdateValuesFile(releaseName string, replicaCount int) error {

	chartPath := hc.Default.OutputDir + "/" + releaseName + "/values.yaml"

	// Read existing values.
	values, err := chartutil.ReadValuesFile(chartPath)
	if err != nil {

		return fmt.Errorf("failed to read values file: %w", err)
	}

	// Modify values.
	valuesMap := values.AsMap()
	valuesMap["replicas"] = replicaCount

	// Write back to file
	data, err := yaml.Marshal(valuesMap)
	if err != nil {

		return fmt.Errorf("failed to marshal values: %w", err)
	}

	return os.WriteFile(chartPath, data, 0644)
}
