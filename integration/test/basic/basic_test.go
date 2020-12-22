// +build k8srequired

package basic

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apptest"
	"github.com/giantswarm/apptest/integration/env"
	"github.com/giantswarm/apptest/integration/setup"
)

var (
	config setup.Config
)

func init() {
	var err error

	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(err.Error())
		}
	}
}

func TestBasic(t *testing.T) {
	var err error

	ctx := context.Background()

	apps := []apptest.App{
		{
			// Install a dependency for the component being tested.
			CatalogName:   "default", // Production catalog.
			Name:          "cert-manager-app",
			Namespace:     metav1.NamespaceSystem,
			Version:       "2.3.1", // Specify the version you need.
			WaitForDeploy: true,
		},
		{
			// Install the component being tested.
			CatalogName:   "control-plane-test-catalog", // Test catalog.
			Name:          "apptest-app",
			Namespace:     "giantswarm",
			SHA:           env.CircleSHA(), // The commit to be tested.
			ValuesYAML:    "e2e: true",     // Provide values for the app.
			WaitForDeploy: true,
		},
	}

	err = config.AppTest.InstallApps(ctx, apps)
	if err != nil {
		t.Fatalf("expected nil got %#q", err)
	}
}
