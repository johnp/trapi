package trapi

import (
	"github.com/coredns/coredns/plugin"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
)

// requestCountAnswered exports a prometheus metric that is incremented every time a query answered by the trapi plugin.
var requestCountAnswered = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: "trapi",
	Name:      "request_count_answered",
	Help:      "Counter of requests answered by this plugin.",
}, []string{"server"})
// TODO: apiCallCount


var once sync.Once
