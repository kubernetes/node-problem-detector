/*
Copyright 2020 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package systemstatsmonitor

import (
	"github.com/golang/glog"
	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/host"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type netCollector struct {
	tags map[string]string

	mNetDevRxBytes      *metrics.Int64Metric
	mNetDevRxPackets    *metrics.Int64Metric
	mNetDevRxErrors     *metrics.Int64Metric
	mNetDevRxDropped    *metrics.Int64Metric
	mNetDevRxFifo       *metrics.Int64Metric
	mNetDevRxFrame      *metrics.Int64Metric
	mNetDevRxCompressed *metrics.Int64Metric
	mNetDevRxMulticast  *metrics.Int64Metric
	mNetDevTxBytes      *metrics.Int64Metric
	mNetDevTxPackets    *metrics.Int64Metric
	mNetDevTxErrors     *metrics.Int64Metric
	mNetDevTxDropped    *metrics.Int64Metric
	mNetDevTxFifo       *metrics.Int64Metric
	mNetDevTxCollisions *metrics.Int64Metric
	mNetDevTxCarrier    *metrics.Int64Metric
	mNetDevTxCompressed *metrics.Int64Metric

	config *ssmtypes.NetStatsConfig
}

func NewNetCollectorOrDie(netConfig *ssmtypes.NetStatsConfig) *netCollector {
	nc := netCollector{tags: map[string]string{}, config: netConfig}

	kernelVersion, err := host.KernelVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve kernel version: %v", err)
	}
	nc.tags[kernelVersionLabel] = kernelVersion

	osVersion, err := util.GetOSVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve OS version: %v", err)
	}
	nc.tags[osVersionLabel] = osVersion

	nc.mNetDevRxBytes, err = metrics.NewInt64Metric(
		metrics.NetDevRxBytes,
		netConfig.MetricsConfigs[string(metrics.NetDevRxBytes)].DisplayName,
		"Cumulative count of bytes received.",
		"Byte",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxBytes, err)
	}

	nc.mNetDevRxPackets, err = metrics.NewInt64Metric(
		metrics.NetDevRxPackets,
		netConfig.MetricsConfigs[string(metrics.NetDevRxPackets)].DisplayName,
		"Cumulative count of packets received.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxPackets, err)
	}

	nc.mNetDevRxErrors, err = metrics.NewInt64Metric(
		metrics.NetDevRxErrors,
		netConfig.MetricsConfigs[string(metrics.NetDevRxErrors)].DisplayName,
		"Cumulative count of receive errors encountered.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxErrors, err)
	}

	nc.mNetDevRxDropped, err = metrics.NewInt64Metric(
		metrics.NetDevRxDropped,
		netConfig.MetricsConfigs[string(metrics.NetDevRxDropped)].DisplayName,
		"Cumulative count of packets dropped while receiving.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxDropped, err)
	}

	nc.mNetDevRxFifo, err = metrics.NewInt64Metric(
		metrics.NetDevRxFifo,
		netConfig.MetricsConfigs[string(metrics.NetDevRxFifo)].DisplayName,
		"Cumulative count of FIFO buffer errors.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxFifo, err)
	}

	nc.mNetDevRxFrame, err = metrics.NewInt64Metric(
		metrics.NetDevRxFrame,
		netConfig.MetricsConfigs[string(metrics.NetDevRxFrame)].DisplayName,
		"Cumulative count of packet framing errors.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxFrame, err)
	}

	nc.mNetDevRxCompressed, err = metrics.NewInt64Metric(
		metrics.NetDevRxCompressed,
		netConfig.MetricsConfigs[string(metrics.NetDevRxCompressed)].DisplayName,
		"Cumulative count of compressed packets received by the device driver.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxCompressed, err)
	}

	nc.mNetDevRxMulticast, err = metrics.NewInt64Metric(
		metrics.NetDevRxMulticast,
		netConfig.MetricsConfigs[string(metrics.NetDevRxMulticast)].DisplayName,
		"Cumulative count of multicast frames received by the device driver.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevRxMulticast, err)
	}

	nc.mNetDevTxBytes, err = metrics.NewInt64Metric(
		metrics.NetDevTxBytes,
		netConfig.MetricsConfigs[string(metrics.NetDevTxBytes)].DisplayName,
		"Cumulative count of bytes transmitted.",
		"Byte",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxBytes, err)
	}

	nc.mNetDevTxPackets, err = metrics.NewInt64Metric(
		metrics.NetDevTxPackets,
		netConfig.MetricsConfigs[string(metrics.NetDevTxPackets)].DisplayName,
		"Cumulative count of packets transmitted.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxPackets, err)
	}

	nc.mNetDevTxErrors, err = metrics.NewInt64Metric(
		metrics.NetDevTxErrors,
		netConfig.MetricsConfigs[string(metrics.NetDevTxErrors)].DisplayName,
		"Cumulative count of transmit errors encountered.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxErrors, err)
	}

	nc.mNetDevTxDropped, err = metrics.NewInt64Metric(
		metrics.NetDevTxDropped,
		netConfig.MetricsConfigs[string(metrics.NetDevTxDropped)].DisplayName,
		"Cumulative count of packets dropped while transmitting.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxDropped, err)
	}

	nc.mNetDevTxFifo, err = metrics.NewInt64Metric(
		metrics.NetDevTxFifo,
		netConfig.MetricsConfigs[string(metrics.NetDevTxFifo)].DisplayName,
		"Cumulative count of FIFO buffer errors.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxFifo, err)
	}

	nc.mNetDevTxCollisions, err = metrics.NewInt64Metric(
		metrics.NetDevTxCollisions,
		netConfig.MetricsConfigs[string(metrics.NetDevTxCollisions)].DisplayName,
		"Cumulative count of collisions detected on the interface.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxCollisions, err)
	}

	nc.mNetDevTxCarrier, err = metrics.NewInt64Metric(
		metrics.NetDevTxCarrier,
		netConfig.MetricsConfigs[string(metrics.NetDevTxCarrier)].DisplayName,
		"Cumulative count of carrier losses detected by the device driver.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxCarrier, err)
	}

	nc.mNetDevTxCompressed, err = metrics.NewInt64Metric(
		metrics.NetDevTxCompressed,
		netConfig.MetricsConfigs[string(metrics.NetDevTxCompressed)].DisplayName,
		"Cumulative count of compressed packets transmitted by the device driver.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, interfaceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.NetDevTxCompressed, err)
	}

	return &nc
}

func (nc *netCollector) recordNetDev() {
	if nc.mNetDevRxBytes == nil {
		return
	}
	if nc.mNetDevRxPackets == nil {
		return
	}
	if nc.mNetDevRxErrors == nil {
		return
	}
	if nc.mNetDevRxDropped == nil {
		return
	}
	if nc.mNetDevRxFifo == nil {
		return
	}
	if nc.mNetDevRxFrame == nil {
		return
	}
	if nc.mNetDevRxCompressed == nil {
		return
	}
	if nc.mNetDevRxMulticast == nil {
		return
	}
	if nc.mNetDevTxBytes == nil {
		return
	}
	if nc.mNetDevTxPackets == nil {
		return
	}
	if nc.mNetDevTxErrors == nil {
		return
	}
	if nc.mNetDevTxDropped == nil {
		return
	}
	if nc.mNetDevTxFifo == nil {
		return
	}
	if nc.mNetDevTxCollisions == nil {
		return
	}
	if nc.mNetDevTxCarrier == nil {
		return
	}
	if nc.mNetDevTxCompressed == nil {
		return
	}

	fs, err := procfs.NewFS("/proc")
	stats, err := fs.NetDev()
	if err != nil {
		glog.Errorf("Failed to retrieve net dev stat: %v", err)
		return
	}

	for iface, ifaceStats := range stats {
		nc.tags[interfaceNameLabel] = iface

		nc.mNetDevRxBytes.Record(nc.tags, int64(ifaceStats.RxBytes))
		nc.mNetDevRxPackets.Record(nc.tags, int64(ifaceStats.RxPackets))
		nc.mNetDevRxErrors.Record(nc.tags, int64(ifaceStats.RxErrors))
		nc.mNetDevRxDropped.Record(nc.tags, int64(ifaceStats.RxDropped))
		nc.mNetDevRxFifo.Record(nc.tags, int64(ifaceStats.RxFIFO))
		nc.mNetDevRxFrame.Record(nc.tags, int64(ifaceStats.RxFrame))
		nc.mNetDevRxCompressed.Record(nc.tags, int64(ifaceStats.RxCompressed))
		nc.mNetDevRxMulticast.Record(nc.tags, int64(ifaceStats.RxMulticast))
		nc.mNetDevTxBytes.Record(nc.tags, int64(ifaceStats.TxBytes))
		nc.mNetDevTxPackets.Record(nc.tags, int64(ifaceStats.TxPackets))
		nc.mNetDevTxErrors.Record(nc.tags, int64(ifaceStats.TxErrors))
		nc.mNetDevTxDropped.Record(nc.tags, int64(ifaceStats.TxDropped))
		nc.mNetDevTxFifo.Record(nc.tags, int64(ifaceStats.TxFIFO))
		nc.mNetDevTxCollisions.Record(nc.tags, int64(ifaceStats.TxCollisions))
		nc.mNetDevTxCarrier.Record(nc.tags, int64(ifaceStats.TxCarrier))
		nc.mNetDevTxCompressed.Record(nc.tags, int64(ifaceStats.TxCompressed))
	}
}

func (nc *netCollector) collect() {
	if nc == nil {
		return
	}

	nc.recordNetDev()
}
