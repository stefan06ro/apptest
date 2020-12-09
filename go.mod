module github.com/giantswarm/apptest

go 1.15

require (
	github.com/giantswarm/apiextensions/v3 v3.13.0
	github.com/giantswarm/appcatalog v0.3.1
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.4.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/prometheus/client_golang v1.7.1 // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/tools v0.0.0-20200616133436-c1934b75d054 // indirect
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/controller-runtime v0.6.3
)

replace sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
