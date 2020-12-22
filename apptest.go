package apptest

import (
	"context"
	"fmt"
	"strings"
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
	failedStatus       = "failed"
	notInstalledStatus = "not-installed"
	defaultNamespace   = "giantswarm"
	uniqueAppCRVersion = "0.0.0"
)

var (
	giantSwarmCatalogs = map[string]string{
		"control-plane-catalog":               "https://giantswarm.github.io/control-plane-catalog/",
		"control-plane-test-catalog":          "https://giantswarm.github.io/control-plane-test-catalog/",
		"default":                             "https://giantswarm.github.io/default-catalog/",
		"default-test":                        "https://giantswarm.github.io/default-test-catalog/",
		"giantswarm":                          "https://giantswarm.github.io/giantswarm-catalog/",
		"giantswarm-test":                     "https://giantswarm.github.io/giantswarm-test-catalog/",
		"giantswarm-operations-platform":      "https://giantswarm.github.io/giantswarm-operations-platform-catalog/",
		"giantswarm-operations-platform-test": "https://giantswarm.github.io/giantswarm-operations-platform-test-catalog/",
		"giantswarm-playground":               "https://giantswarm.github.io/giantswarm-playground-catalog/",
		"giantswarm-playground-test":          "https://giantswarm.github.io/giantswarm-playground-test-catalog/",
		"helm-stable":                         "https://charts.helm.sh/stable/packages/",
		"releases":                            "https://giantswarm.github.io/releases-catalog/",
		"releases-test":                       "https://giantswarm.github.io/releases-test-catalog/",
	}
)

// Config represents the configuration used to setup the apps.
type Config struct {
	KubeConfig     string
	KubeConfigPath string

	Logger micrologger.Logger
	Scheme *runtime.Scheme
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
		if config.Scheme == nil {
			config.Scheme = scheme.Scheme
		}

		// Extend the global client-go scheme which is used by all the tools under
		// the hood. The scheme is required for the controller-runtime controller to
		// be able to watch for runtime objects of a certain type.
		appSchemeBuilder := runtime.SchemeBuilder(schemeBuilder{
			v1alpha1.AddToScheme,
			apiextensionsv1.AddToScheme,
		})
		err = appSchemeBuilder.AddToScheme(config.Scheme)
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

		ctrlClient, err = client.New(rest.CopyConfig(restConfig), client.Options{Scheme: config.Scheme, Mapper: mapper})
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

func (a *AppSetup) UpgradeApp(ctx context.Context, current, desired App) error {
	var err error

	err = a.createAppCatalogs(ctx, []App{current, desired})
	if err != nil {
		return microerror.Mask(err)
	}

	// if current version have no specific version, use the latest instead.
	if current.Version == "" && current.SHA == "" {
		catalogURL, err := getCatalogURL(current)
		if err != nil {
			return microerror.Mask(err)
		}

		version, err := appcatalog.GetLatestVersion(ctx, catalogURL, current.Name, "")
		if err != nil {
			return microerror.Mask(err)
		}

		current.Version = version
	}

	err = a.createApps(ctx, []App{current})
	if err != nil {
		return microerror.Mask(err)
	}

	err = a.waitForDeployedApp(ctx, current)
	if err != nil {
		return microerror.Mask(err)
	}

	err = a.updateApp(ctx, desired)
	if err != nil {
		return microerror.Mask(err)
	}

	err = a.waitForDeployedApp(ctx, desired)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// EnsureCRDs will register the passed CRDs in the k8s API used by the client.
func (a *AppSetup) EnsureCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error {
	var err error
	for _, crd := range crds {
		err = a.ensureCRD(ctx, crd)
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

// CtrlClient returns a controller-runtime client for use in automated tests.
func (a *AppSetup) CtrlClient() client.Client {
	return a.ctrlClient
}

func (a *AppSetup) CleanUp(ctx context.Context, apps []App) error {
	for _, app := range apps {
		err := a.ctrlClient.Delete(ctx, &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: app.AppCRNamespace,
				Name:      app.Name,
			},
			Spec: v1alpha1.AppSpec{},
		})
		if apierrors.IsNotFound(err) {
			// it's ok
		} else if err != nil {
			return microerror.Mask(err)
		}

		var appCRNamespace string
		if app.AppCRNamespace != "" {
			appCRNamespace = app.AppCRNamespace
		} else {
			appCRNamespace = defaultNamespace
		}

		kubeconfigSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: appCRNamespace,
			},
		}
		err = a.ctrlClient.Delete(ctx, kubeconfigSecret)
		if apierrors.IsNotFound(err) {
			// it's ok
		} else if err != nil {
			return microerror.Mask(err)
		}

		userValuesConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: appCRNamespace,
			},
		}
		err = a.ctrlClient.Delete(ctx, userValuesConfigMap)
		if apierrors.IsNotFound(err) {
			// it's ok
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (a *AppSetup) createAppCatalogs(ctx context.Context, apps []App) error {
	for _, app := range apps {
		catalogURL, err := getCatalogURL(app)
		if err != nil {
			return microerror.Mask(err)
		}

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
					URL:  catalogURL,
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

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating %#q app cr from catalog %#q with version %#q", app.Name, app.CatalogName, version))

		var appOperatorVersion string

		if app.AppOperatorVersion != "" {
			appOperatorVersion = app.AppOperatorVersion
		} else {
			// Processed by app-operator-unique instance.
			appOperatorVersion = uniqueAppCRVersion
		}

		var appCRNamespace string

		if app.AppCRNamespace != "" {
			appCRNamespace = app.AppCRNamespace
		} else {
			appCRNamespace = defaultNamespace
		}

		var kubeConfig v1alpha1.AppSpecKubeConfig

		if app.KubeConfig != "" {
			kubeConfigName := fmt.Sprintf("%s-kubeconfig", app.Name)

			err := a.createKubeConfigSecret(ctx, kubeConfigName, appCRNamespace, app.KubeConfig)
			if err != nil {
				return microerror.Mask(err)
			}

			kubeConfig = v1alpha1.AppSpecKubeConfig{
				Context: v1alpha1.AppSpecKubeConfigContext{
					Name: kubeConfigName,
				},
				InCluster: false,
				Secret: v1alpha1.AppSpecKubeConfigSecret{
					Name:      kubeConfigName,
					Namespace: appCRNamespace,
				},
			}
		} else {
			kubeConfig = v1alpha1.AppSpecKubeConfig{
				InCluster: true,
			}
		}

		var userValuesConfigMap string

		if app.ValuesYAML != "" {
			userValuesConfigMap = fmt.Sprintf("%s-user-values", app.Name)

			err := a.createUserValuesConfigMap(ctx, userValuesConfigMap, appCRNamespace, app.ValuesYAML)
			if err != nil {
				return microerror.Mask(err)
			}
		}
		appCR := &v1alpha1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: appCRNamespace,
				Labels: map[string]string{
					label.AppOperatorVersion: appOperatorVersion,
				},
			},
			Spec: v1alpha1.AppSpec{
				Catalog:    app.CatalogName,
				KubeConfig: kubeConfig,
				Name:       app.Name,
				Namespace:  app.Namespace,
				Version:    version,
			},
		}

		if app.ValuesYAML != "" {
			appCR.Spec.UserConfig.ConfigMap.Name = userValuesConfigMap
			appCR.Spec.UserConfig.ConfigMap.Namespace = appCRNamespace
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

func (a *AppSetup) createKubeConfigSecret(ctx context.Context, name, namespace, kubeConfig string) error {
	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating secret '%s/%s'", namespace, name))

	data := map[string][]byte{
		"kubeConfig": []byte(kubeConfig),
	}
	desired := &corev1.Secret{
		Data: data,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := a.k8sClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err := a.k8sClient.CoreV1().Secrets(namespace).Create(ctx, desired, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created secret '%s/%s'", namespace, name))
	} else {
		_, err := a.k8sClient.CoreV1().Secrets(namespace).Update(ctx, desired, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated existing secret '%s/%s'", namespace, name))
	}

	return nil
}

func (a *AppSetup) createUserValuesConfigMap(ctx context.Context, name, namespace, valuesYAML string) error {
	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating '%s/%s' configmap", namespace, name))

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
		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("already created configmap '%s/%s'", namespace, name))
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created configmap '%s/%s'", namespace, name))
	}

	return nil
}

func (a *AppSetup) ensureCRD(ctx context.Context, crd *apiextensionsv1.CustomResourceDefinition) error {
	var err error

	err = a.ctrlClient.Create(ctx, crd)
	if apierrors.IsAlreadyExists(err) {
		// It's ok.
	} else if err != nil {
		return microerror.Mask(err)
	}

	updatedCRD := &apiextensionsv1.CustomResourceDefinition{}

	o := func() error {
		err = a.ctrlClient.Get(ctx, types.NamespacedName{Name: crd.Name}, updatedCRD)
		if err != nil {
			return microerror.Mask(err)
		}

		for _, condition := range updatedCRD.Status.Conditions {
			if condition.Type == "Established" {
				// Fall through.
				return nil
			}
		}

		return microerror.Maskf(executionFailedError, "CRD %#q is not established yet", crd.Name)
	}

	n := func(err error, t time.Duration) {
		a.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get CRD '%s': retrying in %s", crd.Name, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewExponential(1*time.Minute, 10*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (a *AppSetup) updateApp(ctx context.Context, desired App) error {
	var err error
	var currentApp v1alpha1.App

	var appCRNamespace string
	if desired.AppCRNamespace != "" {
		appCRNamespace = desired.AppCRNamespace
	} else {
		appCRNamespace = defaultNamespace
	}

	a.logger.Debugf(ctx, "finding %#q app in namespace %#q", desired.Name, appCRNamespace)

	err = a.ctrlClient.Get(
		ctx,
		types.NamespacedName{Name: desired.Name, Namespace: appCRNamespace},
		&currentApp)
	if err != nil {
		return microerror.Mask(err)
	}

	a.logger.Debugf(ctx, "found %#q app in namespace %#q", desired.Name, appCRNamespace)

	desiredApp := currentApp.DeepCopy()

	var version string
	{
		catalogURL, err := getCatalogURL(desired)
		if err != nil {
			return microerror.Mask(err)
		}

		var appVersion string
		if desired.SHA != "" {
			appVersion = desired.SHA
		} else {
			appVersion = desired.Version
		}

		version, err = appcatalog.GetLatestVersion(ctx, catalogURL, desired.Name, appVersion)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	desiredApp.Spec.Version = version
	desiredApp.Spec.Catalog = desired.CatalogName

	a.logger.Debugf(ctx, "updating %#q app cr in namespace %#q", currentApp.Name, appCRNamespace)
	a.logger.Debugf(ctx, "desired version: %#q", version)
	a.logger.Debugf(ctx, "desired catalog: %#q", desired.CatalogName)

	err = a.ctrlClient.Update(
		ctx,
		desiredApp)
	if err != nil {
		return microerror.Mask(err)
	}

	a.logger.Debugf(ctx, "updated %#q app cr in namespace %#q", currentApp.Name, appCRNamespace)

	return nil
}

func (a *AppSetup) waitForDeployedApps(ctx context.Context, apps []App) error {
	for _, app := range apps {
		if app.WaitForDeploy {
			err := a.waitForDeployedApp(ctx, app)
			if err != nil {
				return microerror.Mask(err)
			}
		} else {
			a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("skipping wait for deploy of %#q app cr", app.Name))
		}
	}

	return nil
}

func (a *AppSetup) waitForDeployedApp(ctx context.Context, testApp App) error {
	var err error

	var appCRNamespace string

	if testApp.AppCRNamespace != "" {
		appCRNamespace = testApp.AppCRNamespace
	} else {
		appCRNamespace = defaultNamespace
	}

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring '%s/%s' app CR is %#q", appCRNamespace, testApp.Name, deployedStatus))

	var app v1alpha1.App

	o := func() error {
		err = a.ctrlClient.Get(
			ctx,
			types.NamespacedName{Name: testApp.Name, Namespace: appCRNamespace},
			&app)
		if err != nil {
			return microerror.Mask(err)
		}

		switch app.Status.Release.Status {
		case notInstalledStatus, failedStatus:
			return backoff.Permanent(microerror.Maskf(executionFailedError, "status %#q, reason: %s", app.Status.Release.Status, app.Status.Release.Reason))
		case deployedStatus:
			if testApp.SHA != "" && strings.HasSuffix(app.Status.Version, testApp.SHA) {
				return nil
			}

			if testApp.Version != "" && testApp.Version == app.Status.Version {
				return nil
			}

			var appVersion string
			if testApp.SHA != "" {
				appVersion = testApp.SHA
			} else {
				appVersion = testApp.Version
			}

			return microerror.Maskf(executionFailedError, "waiting for version contains %#q, current version %#q", appVersion, app.Status.Version)
		}

		return microerror.Maskf(executionFailedError, "waiting for %#q, current %#q", deployedStatus, app.Status.Release.Status)
	}

	n := func(err error, t time.Duration) {
		a.logger.Log("level", "debug", "message", fmt.Sprintf("failed to get app CR status '%s': retrying in %s", deployedStatus, t), "stack", fmt.Sprintf("%v", err))
	}

	b := backoff.NewConstant(20*time.Minute, 10*time.Second)
	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	a.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured '%s/%s' app CR is deployed", appCRNamespace, testApp.Name))

	return nil
}

// getCatalogURL returns the catalog URL for this app. If it is a Giant Swarm
// catalog no URL needs to be provided.
func getCatalogURL(app App) (string, error) {
	if app.CatalogName == "" {
		return "", microerror.Maskf(invalidConfigError, "catalog name must not be empty for app %#v", app)
	}
	if app.CatalogName != "" && app.CatalogURL != "" {
		return app.CatalogURL, nil
	}

	catalogURL, exists := giantSwarmCatalogs[app.CatalogName]
	if !exists {
		return "", microerror.Maskf(invalidConfigError, "catalog %#q not found and no URL provided", app.CatalogName)
	}

	return catalogURL, nil
}

// getVersionForApp checks whether a commit SHA or a version was provided.
// If a SHA was provided then we check the test catalog to get the latest version.
// As for test catalogs the version format used is [latest version]-[sha].
// e.g. 0.2.0-ad12c88111d7513114a1257994634e2ae81115a2
//
// If a version is provided then this is returned. This is to allow app
// dependencies to be installed.
func getVersionForApp(ctx context.Context, app App) (version string, err error) {
	catalogURL, err := getCatalogURL(app)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if app.SHA == "" && app.Version != "" {
		return app.Version, nil
	} else if app.SHA != "" && app.Version == "" {
		version, err := appcatalog.GetLatestVersion(ctx, catalogURL, app.Name, app.SHA)
		if err != nil {
			return "", microerror.Mask(err)
		}

		return version, nil
	} else if app.SHA != "" && app.Version != "" {
		return "", microerror.Maskf(executionFailedError, "both SHA and Version cannot be provided")
	}

	return "", microerror.Maskf(executionFailedError, "either SHA or Version must be provided")
}
