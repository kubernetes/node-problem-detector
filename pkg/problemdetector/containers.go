package problemdetector

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	hostname = os.Getenv("NODE_NAME")
)

type CounterContainer struct {
	counters  map[string]*prometheus.CounterVec
	namespace string
	mutex     sync.Mutex
}

func NewCounterContainer(namespace string) *CounterContainer {
	return &CounterContainer{
		counters:  make(map[string]*prometheus.CounterVec),
		namespace: namespace,
	}
}

func (c *CounterContainer) Fetch(name, help string, labels ...string) (*prometheus.CounterVec, bool) {
	key := containerKey(name, labels)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	counter, exists := c.counters[key]

	if !exists {
		counter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.namespace,
			Name:      name,
			Help:      help,
		}, labels)

		c.counters[key] = counter
	}

	return counter, !exists
}

func containerKey(metric string, labels []string) string {
	s := make([]string, len(labels))
	copy(s, labels)
	sort.Strings(s)
	return fmt.Sprintf("%s{%v}", metric, strings.Join(s, ","))
}
