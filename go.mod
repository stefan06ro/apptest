module github.com/giantswarm/apptest

go 1.15

require (
	github.com/giantswarm/apiextensions/v3 v3.18.0
	github.com/giantswarm/appcatalog v0.4.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
)

replace (
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	// Use v1.4.2 of gorilla/websocket to fix nancy alert.
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	// Use v1.7.1 of viper to fix nancy alert.
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
