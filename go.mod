module github.com/giantswarm/apptest

go 1.16

require (
	github.com/giantswarm/apiextensions/v3 v3.35.0
	github.com/giantswarm/app/v5 v5.3.0
	github.com/giantswarm/appcatalog v0.6.0
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	k8s.io/api v0.20.10
	k8s.io/apiextensions-apiserver v0.20.10
	k8s.io/apimachinery v0.20.10
	k8s.io/client-go v0.20.10
	sigs.k8s.io/controller-runtime v0.6.5
)

replace (
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	github.com/spf13/viper => github.com/spf13/viper v1.7.1
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
