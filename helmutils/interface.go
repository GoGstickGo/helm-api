package helmutils

import (
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

// Logger is an interface that abstracts the logging mechanism.
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Debug(args ...interface{})
}

type InstallAction interface {
	Run(chart *chart.Chart, values map[string]interface{}) (*release.Release, error)
}

type ListAction interface {
	Run() ([]*release.Release, error)
}

type UpgradeAction interface {
	Run(name string, chart *chart.Chart, values map[string]interface{}) (*release.Release, error)
}

type UninstallAction interface {
	Run(name string) (*release.UninstallReleaseResponse, error)
}

type ChartLoader interface {
	Load(path string) (*chart.Chart, error)
}

type Values struct {
	Data map[string]interface{}
}

type FileSystem interface {
	DeleteSubfolder(path string) error
	ReadValuesFile(path string) (*Values, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

type HelmActioner interface {
	NewInstall(config *action.Configuration) InstallAction
	NewList(config *action.Configuration) ListAction
	NewUpgrade(config *action.Configuration) UpgradeAction
	NewUninstall(config *action.Configuration) UninstallAction
}

type RealHelmActioner struct{}

func (h *RealHelmActioner) NewInstall(config *action.Configuration) InstallAction {
	return action.NewInstall(config)
}

func (h *RealHelmActioner) NewUninstall(config *action.Configuration) UninstallAction {
	return action.NewUninstall(config)
}

type Value struct {
	Namespace string
	OutputDir string
	SourceDir string
}

// RealClient is the real implementation of HelmClient using Helm Go SDK.
type RealClient struct {
	ActionConfig *action.Configuration
	Logger       Logger
	Default      Value
	Actioner     HelmActioner
	ChartLoader  ChartLoader
	Filesystem   FileSystem
}

// Configuration struct to hold settings
type Config struct {
	Namespace  string
	OutputDir  string
	SourceDir  string
	HelmDriver string
}
