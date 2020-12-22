// +build k8srequired

package externalcatalog

import (
	"context"
	"testing"

	"github.com/giantswarm/apptest"
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

func TestExternalCatalog(t *testing.T) {
	var err error

	ctx := context.Background()

	apps := []apptest.App{
		{
			// Install app from an external catalog.
			CatalogName:   "flux",
			CatalogURL:    "https://charts.fluxcd.io/", // Specify the catalog URL
			Name:          "flux",
			Namespace:     "giantswarm",
			Version:       "1.5.0", // Specify the version you need.
			WaitForDeploy: true,
		},
	}

	err = config.AppTest.InstallApps(ctx, apps)
	if err != nil {
		t.Fatalf("expected nil got %#q", err)
	}
}
