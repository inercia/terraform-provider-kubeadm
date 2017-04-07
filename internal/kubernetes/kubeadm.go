package kubernetes

import (
	"time"
)

// TypeMeta describes an individual object in an API response or request
// with strings representing the type of the object and its API schema version.
// Structures that are versioned or persisted should inline TypeMeta.
type TypeMeta struct {
	// Kind is a string value representing the REST resource this object represents.
	// Servers may infer this from the endpoint the client submits requests to.
	// Cannot be updated.
	// In CamelCase.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	// +optional
	Kind string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`

	// APIVersion defines the versioned schema of this representation of an object.
	// Servers should convert recognized schemas to the latest internal value, and
	// may reject unrecognized values.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#resources
	// +optional
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,2,opt,name=apiVersion"`
}

type MasterConfiguration struct {
	TypeMeta `json:",inline"`

	API               API        `json:"api,omitempty"`
	Etcd              Etcd       `json:"etcd,omitempty"`
	Networking        Networking `json:"networking,omitempty"`
	KubernetesVersion string     `json:"kubernetesVersion,omitempty"`
	CloudProvider     string     `json:"cloudProvider,omitempty"`
	AuthorizationMode string     `json:"authorizationMode,omitempty"`

	Token    string        `json:"token,omitempty"`
	TokenTTL time.Duration `json:"tokenTTL,omitempty"`

	// SelfHosted enables an alpha deployment type where the apiserver, scheduler, and
	// controller manager are managed by Kubernetes itself. This option is likely to
	// become the default in the future.
	SelfHosted bool `json:"selfHosted"`

	APIServerExtraArgs         map[string]string `json:"apiServerExtraArgs,omitempty"`
	ControllerManagerExtraArgs map[string]string `json:"controllerManagerExtraArgs,omitempty"`
	SchedulerExtraArgs         map[string]string `json:"schedulerExtraArgs,omitempty"`

	// APIServerCertSANs sets extra Subject Alternative Names for the API Server signing cert
	APIServerCertSANs []string `json:"apiServerCertSANs,omitempty"`
	// CertificatesDir specifies where to store or look for all required certificates
	CertificatesDir string `json:"certificatesDir,omitempty"`
}

type API struct {
	// AdvertiseAddress sets the address for the API server to advertise.
	AdvertiseAddress string `json:"advertiseAddress,omitempty"`
	// BindPort sets the secure port for the API Server to bind to
	BindPort int32 `json:"bindPort,omitempty"`
}

type TokenDiscovery struct {
	ID        string   `json:"id,omitempty"`
	Secret    string   `json:"secret,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

type Networking struct {
	ServiceSubnet string `json:"serviceSubnet,omitempty"`
	PodSubnet     string `json:"podSubnet,omitempty"`
	DNSDomain     string `json:"dnsDomain,omitempty"`
}

type Etcd struct {
	Endpoints []string `json:"endpoints,omitempty"`
	CAFile    string   `json:"caFile,omitempty"`
	CertFile  string   `json:"certFile,omitempty"`
	KeyFile   string   `json:"keyFile,omitempty"`
}

type NodeConfiguration struct {
	TypeMeta `json:",inline"`

	CACertPath               string   `json:"caCertPath,omitempty"`
	DiscoveryFile            string   `json:"discoveryFile,omitempty"`
	DiscoveryToken           string   `json:"discoveryToken,omitempty"`
	DiscoveryTokenAPIServers []string `json:"discoveryTokenAPIServers,omitempty"`
	TLSBootstrapToken        string   `json:"tlsBootstrapToken,omitempty"`
	Token                    string   `json:"token,omitempty"`
}
