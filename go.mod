module github.com/inercia/terraform-provider-kubeadm

require (
	github.com/DATA-DOG/go-sqlmock v1.3.3 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/armon/circbuf v0.0.0-20190214190532-5111143e8da2
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/gobuffalo/packr v1.30.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gookit/color v1.1.7
	github.com/hashicorp/terraform v0.12.3
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/mitchellh/go-linereader v0.0.0-20190213213312-1b945b3263eb
	github.com/pkg/errors v0.8.1 // indirect
	github.com/rubenv/sql-migrate v0.0.0-20190717103323-87ce952f7079 // indirect
	github.com/ziutek/mymysql v1.5.4 // indirect
	gopkg.in/gorp.v1 v1.7.2 // indirect
	k8s.io/client-go v0.0.0
	k8s.io/helm v2.16.1+incompatible
	k8s.io/kubernetes v1.16.3
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

// fix some transitional deps
replace github.com/gogo/protobuf v0.0.0-20190410021324-65acae22fc9 => github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d

exclude github.com/Sirupsen/logrus v1.4.1

exclude github.com/Sirupsen/logrus v1.4.0

exclude github.com/Sirupsen/logrus v1.3.0

exclude github.com/Sirupsen/logrus v1.2.0

exclude github.com/Sirupsen/logrus v1.1.1

exclude github.com/Sirupsen/logrus v1.1.0

exclude github.com/renstrom/dedent v1.1.0

// replace some deps for kubernetes v1.16.3 (required by helm 2.16)
replace (
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191114110141-0a35778df828
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191114112024-4bbba8331835
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191114111741-81bb9acf592d
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191114102325-35a9586014f7
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191114112310-0da609c4ca2d
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191114103820-f023614fb9ea
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191114111510-6d1ed697a64b
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191114110717-50a77e50d7d9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191114111229-2e90afcb56c7
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191114110954-d67a8e7e2200
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191114112655-db9be3e678bb
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191114105837-a4a2842dc51b
	k8s.io/node-api => k8s.io/node-api v0.0.0-20191114112948-fde05759caf8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191114104439-68caf20693ac
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.0.0-20191114110435-31b16e91580f
	k8s.io/sample-controller => k8s.io/sample-controller v0.0.0-20191114104921-b2770fad52e3
)

go 1.13
