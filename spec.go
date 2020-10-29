package apptest

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	// InstallApps creates appcatalog and app CRs for use in automated tests
	// and ensures they are installed by our app platform.
	InstallApps(ctx context.Context, apps []App) error

	// K8sClient returns a Kubernetes clienset for use in automated tests.
	K8sClient() kubernetes.Interface
}

type App struct {
	CatalogName   string
	CatalogURL    string
	Name          string
	Namespace     string
	SHA           string
	ValuesYAML    string
	Version       string
	WaitForDeploy bool
}

// schemeBuilder is used to extend the known types of the client-go scheme.
type schemeBuilder []func(*runtime.Scheme) error
