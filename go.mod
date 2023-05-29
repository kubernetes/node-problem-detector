module k8s.io/node-problem-detector

go 1.20

require (
	cloud.google.com/go/compute/metadata v0.2.3
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	contrib.go.opencensus.io/exporter/prometheus v0.0.0-20190427222117-f6cda26f80a3
	contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/acobaugh/osrelease v0.0.0-20181218015638-a93a0a55a249
	github.com/avast/retry-go v2.4.1+incompatible
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/euank/go-kmsg-parser v2.0.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/glog v1.1.1
	github.com/hpcloud/tail v1.0.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.20.1
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_model v0.4.0
	github.com/prometheus/common v0.44.0
	github.com/prometheus/procfs v0.10.0
	github.com/shirou/gopsutil v2.19.12+incompatible
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	go.opencensus.io v0.24.0
	golang.org/x/crypto v0.9.0
	golang.org/x/oauth2 v0.8.0
	golang.org/x/sys v0.8.0
	google.golang.org/api v0.118.0
	k8s.io/api v0.27.2
	k8s.io/apimachinery v0.27.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/component-base v0.27.2
	k8s.io/kubernetes v1.25.10
	k8s.io/test-infra v0.0.0-20190914015041-e1cbc3ccd91c
	k8s.io/utils v0.0.0-20230209194617-a36077c30491
)

require (
	cloud.google.com/go/compute v1.19.0 // indirect
	cloud.google.com/go/container v1.15.0 // indirect
	cloud.google.com/go/monitoring v1.14.0 // indirect
	cloud.google.com/go/trace v1.9.0 // indirect
	github.com/StackExchange/wmi v0.0.0-20181212234831-e0a55b97c705 // indirect
	github.com/aws/aws-sdk-go v1.38.49 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.8.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/s2a-go v0.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.8.0 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.15.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/tedsuo/ifrit v0.0.0-20180802180643-bea94bb476cc // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/term v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace k8s.io/api => k8s.io/api v0.25.10

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.10

replace k8s.io/apimachinery => k8s.io/apimachinery v0.25.10

replace k8s.io/apiserver => k8s.io/apiserver v0.25.10

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.25.10

replace k8s.io/client-go => k8s.io/client-go v0.25.10

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.25.10

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.25.10

replace k8s.io/code-generator => k8s.io/code-generator v0.25.10

replace k8s.io/component-helpers => k8s.io/component-helpers v0.25.10

replace k8s.io/controller-manager => k8s.io/controller-manager v0.25.10

replace k8s.io/cri-api => k8s.io/cri-api v0.25.10

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.25.10

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.25.10

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.25.10

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.10

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.10

replace k8s.io/kubectl => k8s.io/kubectl v0.25.10

replace k8s.io/kubelet => k8s.io/kubelet v0.25.10

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.25.10

replace k8s.io/mount-utils => k8s.io/mount-utils v0.25.10

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.25.10

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.25.10

replace k8s.io/sample-controller => k8s.io/sample-controller v0.25.10

replace k8s.io/component-base => k8s.io/component-base v0.25.10

replace k8s.io/metrics => k8s.io/metrics v0.25.10

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.25.10
