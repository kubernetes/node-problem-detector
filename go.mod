module k8s.io/node-problem-detector

go 1.11

require (
	cloud.google.com/go v0.43.0
	code.cloudfoundry.org/clock v0.0.0-20180518195852-02e53af36e6c
	contrib.go.opencensus.io/exporter/prometheus v0.0.0-20190427222117-f6cda26f80a3
	contrib.go.opencensus.io/exporter/stackdriver v0.12.5
	github.com/StackExchange/wmi v0.0.0-20181212234831-e0a55b97c705 // indirect
	github.com/avast/retry-go v2.4.1+incompatible
	github.com/cobaugh/osrelease v0.0.0-20181218015638-a93a0a55a249
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/euank/go-kmsg-parser v2.0.1+incompatible
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/cadvisor v0.33.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.4.1
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/sigma/go-inotify v0.0.0-20181102212354-c87b6cf5033d // indirect
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.4.0
	github.com/tedsuo/ifrit v0.0.0-20180802180643-bea94bb476cc // indirect
	go.opencensus.io v0.22.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.7.0
	k8s.io/api v0.0.0-20190816222004-e3a6b8045b0b
	k8s.io/apimachinery v0.0.0-20190816221834-a9f1d8a9c101
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/heapster v0.0.0-20180704153620-b25f8a16208f
	k8s.io/kubernetes v1.14.6
	k8s.io/test-infra v0.0.0-20190914015041-e1cbc3ccd91c
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
