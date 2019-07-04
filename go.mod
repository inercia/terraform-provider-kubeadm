module github.com/inercia/terraform-provider-kubeadm

require (
	github.com/chzyer/logex v1.1.11-0.20160617073814-96a4d311aa9b // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/go-cmd/cmd v1.0.4
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gookit/color v1.1.7
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.0 // indirect
	github.com/hashicorp/serf v0.8.2-0.20171022020050-c20a0b1b1ea9 // indirect
	github.com/hashicorp/terraform v0.12.3
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/lithammer/dedent v1.1.0 // indirect
	github.com/miekg/dns v1.0.14 // indirect
	github.com/mitchellh/go-linereader v0.0.0-20190213213312-1b945b3263eb
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/smartystreets/assertions v0.0.0-20190116191733-b6c0e53d7304 // indirect
	github.com/smartystreets/goconvey v0.0.0-20181108003508-044398e4856c // indirect
	github.com/spf13/afero v1.2.2 // indirect
	golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f // indirect
	k8s.io/api v0.0.0-20190626000116-b178a738ed00
	k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // indirect
	k8s.io/apimachinery v0.0.0-20190624085041-961b39a1baa0
	k8s.io/apiserver v0.0.0-20190424053242-2200fef3ea67 // indirect
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider v0.0.0-20190405093944-6c8b65ee8f98 // indirect
	k8s.io/cluster-bootstrap v0.0.0-20190626010831-cd8eb24ea488 // indirect
	k8s.io/kube-proxy v0.0.0-20190314002154-4d735c31b054 // indirect
	k8s.io/kubelet v0.0.0-20190314002251-f6da02f58325 // indirect
	k8s.io/kubernetes v1.14.1
	k8s.io/utils v0.0.0-20190308190857-21c4ce38f2a7 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190626045420-1ec4b74c7bda

exclude github.com/Sirupsen/logrus v1.4.1

exclude github.com/Sirupsen/logrus v1.4.0

exclude github.com/Sirupsen/logrus v1.3.0

exclude github.com/Sirupsen/logrus v1.2.0

exclude github.com/Sirupsen/logrus v1.1.1

exclude github.com/Sirupsen/logrus v1.1.0

exclude github.com/renstrom/dedent v1.1.0
