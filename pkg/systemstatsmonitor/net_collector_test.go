package systemstatsmonitor

import (
	"os"
	"path"
	"regexp"
	"testing"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

var defaultMetricsConfig = map[string]ssmtypes.MetricConfig{
	string(metrics.NetDevRxBytes):      {DisplayName: string(metrics.NetDevRxBytes)},
	string(metrics.NetDevRxPackets):    {DisplayName: string(metrics.NetDevRxPackets)},
	string(metrics.NetDevRxErrors):     {DisplayName: string(metrics.NetDevRxErrors)},
	string(metrics.NetDevRxDropped):    {DisplayName: string(metrics.NetDevRxDropped)},
	string(metrics.NetDevRxFifo):       {DisplayName: string(metrics.NetDevRxFifo)},
	string(metrics.NetDevRxFrame):      {DisplayName: string(metrics.NetDevRxFrame)},
	string(metrics.NetDevRxCompressed): {DisplayName: string(metrics.NetDevRxCompressed)},
	string(metrics.NetDevRxMulticast):  {DisplayName: string(metrics.NetDevRxMulticast)},
	string(metrics.NetDevTxBytes):      {DisplayName: string(metrics.NetDevTxBytes)},
	string(metrics.NetDevTxPackets):    {DisplayName: string(metrics.NetDevTxPackets)},
	string(metrics.NetDevTxErrors):     {DisplayName: string(metrics.NetDevTxErrors)},
	string(metrics.NetDevTxDropped):    {DisplayName: string(metrics.NetDevTxDropped)},
	string(metrics.NetDevTxFifo):       {DisplayName: string(metrics.NetDevTxFifo)},
	string(metrics.NetDevTxCollisions): {DisplayName: string(metrics.NetDevTxCollisions)},
	string(metrics.NetDevTxCarrier):    {DisplayName: string(metrics.NetDevTxCarrier)},
	string(metrics.NetDevTxCompressed): {DisplayName: string(metrics.NetDevTxCompressed)},
}

// To get a similar output, run `cat /proc/net/dev` on a Linux machine
// docker:	1500	100		8		7		0		0		0		0		9000	450		565		200		20		30		0		0
const fakeNetProcContent = `Inter-|   Receive                                                |  Transmit
face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
eth0:		5000	100		0		0		0 		0 		0 		0		2500	30		0		0		0		0		0		0  
docker0: 	1000	90		8		7		0 		0 		0 		0		0		0		0		0		0		0		0		0
docker1: 	500		10		0		0		0 		0		0		0		3000	150		15		0		20		30		0		0
docker2:	0		0		0		0		0		0		0		0		6000	300		550		200		0		0		0		0
`

// newFakeInt64Metric is a wrapper around metrics.NewFakeInt64Metric
func newFakeInt64Metric(metricID metrics.MetricID, viewName string, description string, unit string, aggregation metrics.Aggregation, tagNames []string) (metrics.Int64MetricInterface, error) {
	return metrics.NewFakeInt64Metric(viewName, aggregation, tagNames), nil
}

// testCollectAux is a test auxiliary function used for testing netCollector.Collect
func testCollectAux(t *testing.T, name string, excludeInterfaceRegexp ssmtypes.NetStatsInterfaceRegexp, validate func(*testing.T, *netCollector)) {
	// mkdir /tmp/proc-X
	procDir := t.TempDir()

	// mkdir -C /tmp/proc-X/net
	procNetDir := path.Join(procDir, "net")
	if err := os.Mkdir(procNetDir, 0777); err != nil {
		t.Fatalf("Failed to create directory %q: %v", procNetDir, err)
	}

	// touch /tmp/proc-X/net/dev
	filename := path.Join(procNetDir, "dev")
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create file %q: %v", filename, err)
	}
	// echo $FILE_CONTENT > /tmp/proc-X/net/dev
	if _, err = f.WriteString(fakeNetProcContent); err != nil {
		t.Fatalf("Failed to write to file %q: %v", filename, err)
	}
	if err = f.Close(); err != nil {
		t.Fatalf("Failed to close file %q: %v", filename, err)
	}

	// Build the netCollector
	config := &ssmtypes.NetStatsConfig{
		ExcludeInterfaceRegexp: excludeInterfaceRegexp,
		MetricsConfigs:         defaultMetricsConfig,
	}
	netCollector := &netCollector{
		config:   config,
		procPath: procDir,
		recorder: newIfaceStatRecorder(newFakeInt64Metric),
	}
	netCollector.initOrDie()
	netCollector.collect()
	validate(t, netCollector)
}

func TestCollect(t *testing.T) {
	tcs := []struct {
		Name                   string
		ExcludeInterfaceRegexp ssmtypes.NetStatsInterfaceRegexp
		Validate               func(t *testing.T, nc *netCollector)
	}{
		{
			Name:                   "NoFilterMatch",
			ExcludeInterfaceRegexp: ssmtypes.NetStatsInterfaceRegexp{R: regexp.MustCompile(`^fake$`)},
			Validate: func(t *testing.T, nc *netCollector) {
				// We just validate two metrics, no need to check all of them
				expectedValues := map[metrics.MetricID]map[string]int64{
					metrics.NetDevRxBytes: {
						"eth0":    5000,
						"docker0": 1000,
						"docker1": 500,
						"docker2": 0,
					},
					metrics.NetDevTxBytes: {
						"eth0":    2500,
						"docker0": 0,
						"docker1": 3000,
						"docker2": 6000,
					},
				}
				for metricID, interfaceValues := range expectedValues {
					collector, ok := nc.recorder.collectors[metricID]
					if !ok {
						t.Errorf("Failed to get collector of metric %s", metricID)
						continue
					}
					fakeInt64Metric, ok := collector.metric.(*metrics.FakeInt64Metric)
					if !ok {
						t.Fatalf("Failed to convert metric %s to fakeMetric", string(metricID))
					}
					for _, repr := range fakeInt64Metric.ListMetrics() {
						interfaceName, ok := repr.Labels[interfaceNameLabel]
						if !ok {
							t.Fatalf("Failed to get label %q for ", interfaceNameLabel)
						}
						expectedValue, ok := interfaceValues[interfaceName]
						if !ok {
							t.Fatalf("Failed to get metric value for interface %q", interfaceName)
						}
						if repr.Value != expectedValue {
							t.Errorf("Mismatch in metric %q for interface %q: expected %d, got %d", metricID, interfaceName, expectedValue, repr.Value)
						}
					}
				}
			},
		},
		{
			Name:                   "FilterMatch",
			ExcludeInterfaceRegexp: ssmtypes.NetStatsInterfaceRegexp{R: regexp.MustCompile(`docker\d+`)},
			Validate: func(t *testing.T, nc *netCollector) {
				// We just validate two metrics, no need to check all of them
				expectedValues := map[metrics.MetricID]map[string]int64{
					metrics.NetDevRxBytes: {
						"eth0": 5000,
					},
					metrics.NetDevTxBytes: {
						"eth0": 2500,
					},
				}
				for metricID, interfaceValues := range expectedValues {
					collector, ok := nc.recorder.collectors[metricID]
					if !ok {
						t.Errorf("Failed to get collector of metric %s", metricID)
						continue
					}
					fakeInt64Metric, ok := collector.metric.(*metrics.FakeInt64Metric)
					if !ok {
						t.Fatalf("Failed to convert metric %s to fakeMetric", string(metricID))
					}
					for _, repr := range fakeInt64Metric.ListMetrics() {
						interfaceName, ok := repr.Labels[interfaceNameLabel]
						if !ok {
							t.Fatalf("Failed to get label %q for ", interfaceNameLabel)
						}
						expectedValue, ok := interfaceValues[interfaceName]
						if !ok {
							t.Fatalf("Failed to get metric value for interface %q", interfaceName)
						}
						if repr.Value != expectedValue {
							t.Errorf("Mismatch in metric %q for interface %q: expected %d, got %d", metricID, interfaceName, expectedValue, repr.Value)
						}
					}
				}
			},
		},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			testCollectAux(t, tc.Name, tc.ExcludeInterfaceRegexp, tc.Validate)
		})
	}
}
