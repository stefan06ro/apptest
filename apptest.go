package apptest

import (
	"context"
	"fmt"
	"time"

	v1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	deployedStatus     = "deployed"
	namespace          = "giantswarm"
	uniqueAppCRVersion = "0.0.0"
)

// Config represents the configuration used to setup the apps.
type Config struct {
	KubeConfig     string
	KubeConfigPath string

	Logger micrologger.Logger
}

// AppSetup implements the logic for managing the app setup.
type AppSetup struct {
	ctrlClient client.Client
	k8sClient  kubernetes.Interface
	logger     micrologger.Logger
}

// New creates a new configured app setup library.
func New(config Config) (*AppSetup, error) {
	var err error

	if config.KubeConfig == "" && config.KubeConfigPath == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.KubeConfig and %T.KubeConfigPath must not be empty at the same time", config, config)
	}
	if config.KubeConfig != "" && config.KubeConfigPath != "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.KubeConfig and %T.KubeConfigPath must not be set at the same time", config, config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var restConfig *rest.Config
	{
		if config.KubeConfig != "" {
			bytes := []byte(config.KubeConfig)
			restConfig, err = clientcmd.RESTConfigFromKubeConfig(bytes)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		} else if config.KubeConfigPath != "" {
			restConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		} else {
			// Shouldn't happen but returning error just in case.
			return nil, microerror.Maskf(invalidConfigError, "%T.KubeConfig and %T.KubeConfigPath must not be empty at the same time", config, config)
		}
	}

	var ctrlClient client.Client
	{
		// Extend the global client-go scheme which is used by all the tools under
		// the hood. The scheme is required for the controller-runtime controller to
		// be able to watch for runtime objects of a certain type.
		appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
			v1alpha1.AddToScheme,
			apiextensionsv1.AddToScheme,
		})
		err = appSchemeBuilder.AddToScheme(scheme.Scheme)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Configure a dynamic rest mapper to the controller client so it can work
		// with runtime objects of arbitrary types. Note that this is the default
		// for controller clients created by controller-runtime managers.
		// Anticipating a rather uncertain future and more breaking changes to come
		// we want to separate client and manager. Thus we configure the client here
		// properly on our own instead of relying on the manager to provide a
		// client, which might change in the future.
		mapper, err := apiutil.NewDynamicRESTMapper(rest.CopyConfig(restConfig))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		ctrlClient, err = client.New(rest.CopyConfig(restConfig), client.Options{Scheme: scheme.Scheme, Mapper: mapper})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var k8sClient kubernetes.Interface
	{
		c := rest.CopyConfig(restConfig)

		k8sClient, err = kubernetes.NewForConfig(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	a := &AppSetup{
		ctrlClient: ctrlClient,
		k8sClient:  k8sClient,
		logger:     config.Logger,
	}

	return a, nil
}

// InstallApps creates appcatalog and app CRs for use in automated tests
// and ensures they are installed by our app platform.
func (a *AppSetup) InstallApps(ctx context.Context, apps []App) error {
	var err error

	err = a.createAppCatalogs(ctx, apps)
	if err != nil {
		return microerror.Mask(err)
	}

	err = a.createApps(ctx, apps)
	if err != nil {
		return microerror.Mask(err)
	}

	err = a.waitForDeployedApps(ctx, apps)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// EnsureCRDs will register the passed CRDs in the k8s API used by the client.
func (a *AppSetup) EnsureCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error {
	var err error
	for _, crd := range crds {
		err = a.ctrlClient.Create(ctx, crd)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// K8sClient returns a Kubernetes clienset for use in automated tests.
func (a *AppSetup) K8sClient() kubernetes.Interface {
	return a.k8sClient
}

func (a *AppSetup) createAppCatalogs(ctx context.Context, apps []App) error {
	var err error

	for _, app := range apps {
		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q appcatalog cr", app.CatalogName))

		appCatalogCR := &v1alpha1.AppCatalog{
			ObjectMeta: metav1.ObjectMeta{
				Name: app.CatalogName,
				Labels: map[string]string{
					// Processed by app-operator-unique.
					label.AppOperatorVersion: uniqueAppCRVersion,
				},
			},
			Spec: v1alpha1.AppCatalogSpec{
				Description: app.CatalogName,
				Title:       app.CatalogName,
				Storage: v1alpha1.AppCatalogSpecStorage{
					Type: "helm",
					URL:  app.CatalogURL,
				},
			},
		}
		err = a.ctrlClient.Create(ctx, appCatalogCR)
		if apierrors.IsAlreadyExists(err) {
			a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q appcatalog CR already exists", appCatalogCR.Name))
		} else if err != nil {
			return microerror.Mask(err)
		}

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %#q appcatalog cr", app.CatalogName))
	}

	return nil
}

func (a *AppSetup) createApps(ctx context.Context, apps []App) error {
	for _, app := range apps {
		// Get app version based on whether a commit SHA or a version was
		// provided.
		version, err := getVersionForApp(ctx, app)
		if err != nil {
			return microerror.Mask(err)
		}

		var userValuesConfigMap string

		if app.ValuesYAML != "" {
			userValuesConfigMap = fmt.Sprintf("%s-user-values", app.Name)

			err := a.createUserValuesConfigMap(ctx, userValuesConfigMap, namespace, app.ValuesYAML)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q app cr from catalog %#q with version %#q", app.Name, app.CatalogName, version))

		appCR := &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: namespace,
				Labels: map[string]string{
					// Processed by app-operator-unique.
					label.AppOperatorVersion: uniqueAppCRVersion,
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog: app.CatalogName,
				KubeConfig: v1alpha1.AppSpecKubeConfig{
					InCluster: true,
				},
				Name:      app.Name,
				Namespace: app.Namespace,
				Version:   version,
			},
		}

		if userValuesConfigMap != "" {
			appCR.Spec.UserConfig.ConfigMap.Name = userValuesConfigMap
			appCR.Spec.UserConfig.ConfigMap.Namespace = namespace
		}

		err = a.ctrlClient.Create(ctx, appCR)
		if apierrors.IsAlreadyExists(err) {
			a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q app CR already exists", appCR.Name))
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created %#q app cr", appCR.Name))
	}

	return nil
}

func (a *AppSetup) createUserValuesConfigMap(ctx context.Context, name, namespace, valuesYAML string) error {
	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q configmap", name))

	values := map[string]string{
		"values": valuesYAML,
	}
	configMap := &corev1.ConfigMap{
		Data: values,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := a.k8sClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created configmap %#q", name))
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created configmap %#q", name))
	}

	return nil
}

func (a *AppSetup) waitForDeployedApps(ctx context.Context, apps []App) error {
	for _, app := range apps {
		if app.WaitForDeploy {
			err := a.waitForDeployedApp(ctx, app.Name)
			if err != nil {
				return microerror.Mask(err)
			}
		} else {
			a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("skipping wait for deploy of %#q app cr", app.Name))
		}
	}

	return nil
}

func (a *AppSetup) waitForDeployedApp(ctx context.Context, appName string) error {
	var err error

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring %#q app CR is %#q", appName, deployedStatus))

	var app v1alpha1.App

	o := func() error {
		err = a.ctrlClient.Get(
			ctx,
			types.NamespacedName{Name: appName, Namespace: namespace},
			&app)
		if err != nil {
			return microerror.Mask(err)
		}
		if app.Status.Release.Status != deployedStatus {
			return microerror.Maskf(executionFailedError, "waiting for %#q, current %#q", deployedStatus, app.Status.Release.Status)
		}

		return nil
	}

	n := func(err error, t time.Duration) {
		a.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get app CR status '%s': retrying in %s", deployedStatus, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewConstant(20*time.Minute, 10*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured %#q app CR is deployed", appName))

	return nil
}

// getVersionForApp checks whether a commit SHA or a version was provided.
// If a SHA was provided then we check the test catalog to get the latest version.
// As for test catalogs the version format used is [latest version]-[sha].
// e.g. 0.2.0-ad12c88111d7513114a1257994634e2ae81115a2
//
// If a version is provided then this is returned. This is to allow app
// dependencies to be installed.
func getVersionForApp(ctx context.Context, app App) (version string, err error) {
	if app.SHA == "" && app.Version != "" {
		return app.Version, nil
	} else if app.SHA != "" && app.Version == "" {
		version, err := appcatalog.GetLatestVersion(ctx, app.CatalogURL, app.Name, "")
		if err != nil {
			return "", microerror.Mask(err)
		}

		return version, nil
	} else if app.SHA != "" && app.Version != "" {
		return "", microerror.Maskf(executionFailedError, "both SHA and Version cannot be provided")
	}

	return "", microerror.Maskf(executionFailedError, "either SHA or Version must be provided")
}
