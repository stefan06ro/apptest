[![GoDoc](https://godoc.org/github.com/giantswarm/apptest?status.svg)](http://godoc.org/github.com/giantswarm/apptest) [![CircleCI](https://circleci.com/gh/giantswarm/apptest.svg?&style=shield)](https://circleci.com/gh/giantswarm/apptest)

# apptest

Go library for using the Giant Swarm app platform in integration tests.

## Design Goals

- apptest should have minimal Go dependencies.
- apptest should be useable in any cluster but [kind] (Kubernetes in Docker) is
the primary target for local development. 
- [apptestctl] bootstraps app platform and has complex dependencies but is used
as a downloadble CLI.
- Components are installed via [app CR] to match how we deploy them in production.

## Setup

- apptest is designed to be used with the [integration-test] job in `architect-orb`.
- `install-app-platform` must be true and triggers an `apptestctl bootstrap`.

```yaml
version: 2.1

orbs:
  architect: giantswarm/architect@1.0.0

workflows:
  test:
    jobs:
    - architect/integration-test:
        name: "basic-integration-test"
        install-app-platform: true
        test-dir: "integration/test/basic"
        requires:
          - push-test-app-to-control-plane-catalog
```

## K8s Clients and CRDs

- Two clients are exposed to be used to interact with the cluster during tests.
- The `CtrlClient` allows you to interact with custom resources but the CRDs
need to be installed with `EnsureCRDs` or via a Helm chart.
- Only CRDs in our [apiextensions] library can be used with `EnsureCRDs`.

```go
import (
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	// CtrlClient returns a controller-runtime client for use in automated tests.
	CtrlClient() client.Client

	// EnsureCRDs will register the passed CRDs in the k8s API used by the client.
	EnsureCRDs(ctx context.Context, crds []*apiextensionsv1.CustomResourceDefinition) error

	// K8sClient returns a Kubernetes clienset for use in automated tests.
	K8sClient() kubernetes.Interface
}
``` 

## Running Tests Locally

- The steps below can be used to run tests locally with [kind].

```sh
kind create cluster

apptestctl bootstrap --kubeconfig="$(kind get kubeconfig)"

kind get kubeconfig > /tmp/kind-kubeconfig

export E2E_KUBECONFIG=/tmp/kind-kubeconfig
export CIRCLE_SHA1=$(git rev-parse HEAD)

go test -v -tags=k8srequired ./integration/test/basic -count=1 | luigi
```

Note:

To test the Helm chart of the app and any related binaries you need to
have pushed your changes to GitHub. We want to be able to test all local changes
but this is not yet supported.

## Examples

These examples show common use cases for using apptest in automated tests.
Each example also has an integration test you can refer to.

###  Basic test

Install the component being tested and optionally any other apps it depends on.

Test: [basic-test]

```go
apps := []apptest.App{
  {
    // Install a dependency for the component being tested.
    CatalogName:   "control-plane-catalog", // Production catalog.
    Name:          "cert-manager-app",
    Namespace:     metav1.NamespaceSystem,
    Version:       "2.3.1", // Specify the version you need.
    WaitForDeploy: true,
  },
  {
    // Install the component being tested.
    CatalogName:   "control-plane-test-catalog", // Test catalog.
    Name:          "app-admission-controller",
    Namespace:     "giantswarm",
    SHA:           env.CircleSHA(), // The commit to be tested.
    ValuesYAML:    "e2e: true", // Provide values for the app.
    WaitForDeploy: true,
  },
}
err = appTest.InstallApps(ctx, apps)
if err != nil {
  t.Fatalf("expected nil got %#q", err)
}
```

## Ensure CRDs

Install a CRD from our [apiextensions] library for use in a test.

Test: [ensure-crds-test]

```go
import (
	monitoringv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/monitoring/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/crd"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Add the extra CRDs you need to the scheme.
appSchemeBuilder := runtime.SchemeBuilder{
  monitoringv1alpha1.AddToScheme,
}
err = appSchemeBuilder.AddToScheme(runtimeScheme)
if err != nil {
  t.Fatalf("expected nil got %#q", err)
}

c := apptest.Config{
  Logger: config.Logger,
  Scheme: runtimeScheme,

  KubeConfigPath: env.KubeConfigPath(),
}

appTest, err := apptest.New(c)
if err != nil {
  t.Fatalf("expected nil got %#q", err)
}

// Define the CRDs you wish to install.
crds := []*apiextensionsv1.CustomResourceDefinition {
  crd.LoadV1("cluster.x-k8s.io", "Cluster"),
}

err = appTest.EnsureCRDs(ctx, crds)
if err != nil {
    t.Fatalf("expected nil got %#q", err)
}
```

## External catalog

A list of known Giant Swarm catalogs is maintained in apptest to avoid needing
to set the catalog URL. But installing from external catalogs is also possible.

Test: [external-catalog-test]

```go
{
  apps := []apptest.App{
    {
      // Install app from an external catalog. 
      CatalogName:   "flux", 
      CatalogURL:   "https://charts.fluxcd.io/", // Specify the catalog URL
      Name:          "flux",
      Namespace:     "giantswarm",
      Version:       "1.5.0", // Specify the version you need.
      WaitForDeploy: true,
    },
  }
  err = appTest.InstallApps(ctx, apps)
  if err != nil {
    t.Fatalf("expected nil got %#q", err)
  }
}
```

[app CR]: https://docs.giantswarm.io/reference/cp-k8s-api/apps.application.giantswarm.io/
[apiextensions]: https://github.com/giantswarm/apiextensions
[apptestctl]: https://github.com/giantswarm/apptestctl
[client-go]: https://github.com/kubernetes/client-go 
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[integration-test-job]: https://github.com/giantswarm/architect-orb/blob/master/docs/job/integration-test.md
[kind]: https://kind.sigs.k8s.io/

[basic-test]: https://github.com/giantswarm/apptest/tree/master/integration/test/basic/basic.go
[ensure-crds-test]: https://github.com/giantswarm/apptest/tree/master/integration/test/ensurecrds/ensure_crds.go
[external-catalog-test]: https://github.com/giantswarm/apptest/tree/master/integration/test/externalcatalog/external_catalog.go
