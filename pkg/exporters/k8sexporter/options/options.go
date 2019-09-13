package options

import (
	"time"

	"github.com/spf13/pflag"
)

type CommandLineOptions struct {
	// EnableK8sExporter is the flag determining whether to report to Kubernetes.
	EnableK8sExporter bool
	// HostnameOverride specifies custom node name used to override hostname.
	HostnameOverride string
	// ApiServerOverride is the custom URI used to connect to Kubernetes ApiServer.
	ApiServerOverride string
	// APIServerWaitTimeout is the timeout on waiting for kube-apiserver to be
	// ready.
	APIServerWaitTimeout time.Duration
	// APIServerWaitInterval is the interval between the checks on the
	// readiness of kube-apiserver.
	APIServerWaitInterval time.Duration
	// K8sExporterHeartbeatPeriod is the period at which the k8s exporter does forcibly sync with apiserver.
	K8sExporterHeartbeatPeriod time.Duration
	// ServerPort is the port to bind the node problem detector server. Use 0 to disable.
	ServerPort int
	// ServerAddress is the address to bind the node problem detector server.
	ServerAddress string
}

func (o *CommandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.EnableK8sExporter, "enable-k8s-exporter", true,
		"Enables reporting to Kubernetes API server.")
	fs.StringVar(&o.ApiServerOverride, "apiserver-override", "",
		"Custom URI used to connect to Kubernetes ApiServer. This is ignored if --enable-k8s-exporter is false.")
	fs.DurationVar(&o.APIServerWaitTimeout, "apiserver-wait-timeout", time.Duration(5)*time.Minute,
		"The timeout on waiting for kube-apiserver to be ready. This is ignored if --enable-k8s-exporter is false.")
	fs.DurationVar(&o.APIServerWaitInterval, "apiserver-wait-interval", time.Duration(5)*time.Second,
		"The interval between the checks on the readiness of kube-apiserver. This is ignored if --enable-k8s-exporter is false.")
	fs.DurationVar(&o.K8sExporterHeartbeatPeriod, "k8s-exporter-heartbeat-period", time.Duration(5)*time.Minute,
		"The period at which k8s-exporter does forcibly sync with apiserver.")
	fs.StringVar(&o.HostnameOverride, "hostname-override", "",
		"Custom node name used to override hostname")
	fs.IntVar(&o.ServerPort, "port", 20256,
		"The port to bind the node problem detector server. Use 0 to disable.")
	fs.StringVar(&o.ServerAddress, "address", "127.0.0.1",
		"The address to bind the node problem detector server.")
}
