package kmsg

import (
	"k8s.io/klog/v2"
)

// kmsgParserLogger is compatible with the kmsgparser.Logger interface
// redirecting the log messages to klog
type kmsgParserLogger struct{}

func (k *kmsgParserLogger) Infof(format string, args ...interface{}) {
	klog.Infof(format, args...)
}

func (k *kmsgParserLogger) Errorf(format string, args ...interface{}) {
	klog.Errorf(format, args...)
}

func (k *kmsgParserLogger) Warningf(format string, args ...interface{}) {
	klog.Warningf(format, args...)
}
