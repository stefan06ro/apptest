package apptest

import "context"

type Interface interface {
	// InstallApps creates appcatalog and app CRs for use in automated tests
	// and ensures they are installed by our app platform.
	InstallApps(ctx context.Context, apps []App) error
}

type App struct {
	CatalogName   string
	CatalogURL    string
	Name          string
	Namespace     string
	Version       string
	WaitForDeploy bool
}
