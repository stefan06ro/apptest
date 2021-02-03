package apptest

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	// InstallApps creates appcatalog and app CRs for use in automated tests
	// and ensures they are installed by our app platform.
	InstallApps(ctx context.Context, apps []App) error

	// UpgradeApp find matching current app CR and change the spec
	// to follow desired app CR.
	UpgradeApp(ctx context.Context, current, desired App) error

	// EnsureCRDs will register the passed CRDs in the k8s API used by the client.
	EnsureCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error

	// K8sClient returns a Kubernetes clienset for use in automated tests.
	K8sClient() kubernetes.Interface

	// CtrlClient returns a controller-runtime client for use in automated tests.
	CtrlClient() client.Client

	// CleanUp removes created resources while installing apps.
	CleanUp(ctx context.Context, apps []App) error
}

type App struct {
	AppCRName          string
	AppCRNamespace     string
	AppOperatorVersion string
	CatalogName        string
	CatalogURL         string
	KubeConfig         string
	Name               string
	Namespace          string
	SHA                string
	ValuesYAML         string
	Version            string
	WaitForDeploy      bool
}

// schemeBuilder is used to extend the known types of the client-go scheme.
type schemeBuilder []func(*runtime.Scheme) error
