// Code generated automatically with go generate; DO NOT EDIT.

package assets

const CloudProviderCode = `# from https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/

{{- if .cloud_config}}
apiVersion: v1
kind: Secret
metadata:
  name: cloud-provider-config
type: Opaque
data:
  # "cloud_config" contains the Base64 encoded configuration file
  cloud.conf: {{.cloud_config}}
{{- end}}

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloud-controller-manager
  namespace: kube-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:cloud-controller-manager
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: cloud-controller-manager
    namespace: kube-system

---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    k8s-app: cloud-controller-manager
  name: cloud-controller-manager
  namespace: kube-system
spec:
  selector:
    matchLabels:
      k8s-app: cloud-controller-manager
  template:
    metadata:
      labels:
        k8s-app: cloud-controller-manager
    spec:
      serviceAccountName: cloud-controller-manager
      containers:
        - name: cloud-controller-manager
          # for in-tree providers we use k8s.gcr.io/cloud-controller-manager
          # this can be replaced with any other image for out-of-tree providers
          image: k8s.gcr.io/cloud-controller-manager:v1.8.0
          command:
            - /usr/local/bin/cloud-controller-manager
            - --cloud-provider={{.cloud_provider}}
{{- if .cloud_config}}
            - --cloud-config=/etc/kubernetes/cloud/cloud.conf
{{- end}}
            - --leader-elect=true
            - --use-service-account-credentials
            # these flags will vary for every cloud provider
{{- if .cloud_provider_flags}}
            - {{.cloud_provider_flags}}
{{- end}}
{{- if .cloud_config}}
          volumeMounts:
            - name: cloud-provider-config
              mountPath: "/etc/kubernetes/cloud"
{{- end}}

      tolerations:
        # this is required so CCM can bootstrap itself
        - key: node.cloudprovider.kubernetes.io/uninitialized
          value: "true"
          effect: NoSchedule
        # this is to have the daemonset runnable on master nodes
        # the taint may vary depending on your cluster setup
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
      # this is to restrict CCM to only run on master nodes
      # the node selector may vary depending on your cluster setup
      nodeSelector:
        node-role.kubernetes.io/master: ""

{{- if .cloud_config}}
      volumes:
        - name: cloud-provider-config
          secret:
            secretName: cloud-provider-config
{{- end}}
`
