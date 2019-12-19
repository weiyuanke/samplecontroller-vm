/**
from prometheus kubernetes client_metrics
 */
package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/util/workqueue"
)

const workqueueMetricsNamespace = "myWorkqueue"

var (
	clientGoWorkqueueDepthMetricVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: workqueueMetricsNamespace,
			Name:      "depth",
			Help:      "Current depth of the work queue.",
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueAddsMetricVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: workqueueMetricsNamespace,
			Name:      "items_total",
			Help:      "Total number of items added to the work queue.",
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueLatencyMetricVec = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  workqueueMetricsNamespace,
			Name:       "latency_seconds",
			Help:       "How long an item stays in the work queue.",
			Objectives: map[float64]float64{},
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueUnfinishedWorkSecondsMetricVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: workqueueMetricsNamespace,
			Name:      "unfinished_work_seconds",
			Help:      "How long an item has remained unfinished in the work queue.",
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueLongestRunningProcessorMetricVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: workqueueMetricsNamespace,
			Name:      "longest_running_processor_seconds",
			Help:      "Duration of the longest running processor in the work queue.",
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueWorkDurationMetricVec = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  workqueueMetricsNamespace,
			Name:       "work_duration_seconds",
			Help:       "How long processing an item from the work queue takes.",
			Objectives: map[float64]float64{},
		},
		[]string{"queue_name"},
	)
	clientGoWorkqueueRetrysMetricVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: workqueueMetricsNamespace,
			Name:      "retry_total",
			Help:      "Total number of retry times.",
		},
		[]string{"queue_name"},
	)
)

// Definition of client-go workqueue metrics provider definition
type workqueueCustomMetricsProvider struct{}

func (f *workqueueCustomMetricsProvider) Register(registerer prometheus.Registerer) {
	workqueue.SetProvider(f)
	registerer.MustRegister(
		clientGoWorkqueueDepthMetricVec,
		clientGoWorkqueueAddsMetricVec,
		clientGoWorkqueueLatencyMetricVec,
		clientGoWorkqueueWorkDurationMetricVec,
		clientGoWorkqueueUnfinishedWorkSecondsMetricVec,
		clientGoWorkqueueLongestRunningProcessorMetricVec,
	)
}

func (f *workqueueCustomMetricsProvider) NewDepthMetric(name string) workqueue.GaugeMetric {
	return clientGoWorkqueueDepthMetricVec.WithLabelValues(name)
}
func (f *workqueueCustomMetricsProvider) NewAddsMetric(name string) workqueue.CounterMetric {
	return clientGoWorkqueueAddsMetricVec.WithLabelValues(name)
}
func (f *workqueueCustomMetricsProvider) NewLatencyMetric(name string) workqueue.HistogramMetric {
	metric := clientGoWorkqueueLatencyMetricVec.WithLabelValues(name)
	// Convert microseconds to seconds for consistency across metrics.
	return prometheus.ObserverFunc(func(v float64) {
		metric.Observe(v / 1e6)
	})
}
func (f *workqueueCustomMetricsProvider) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	metric := clientGoWorkqueueWorkDurationMetricVec.WithLabelValues(name)
	// Convert microseconds to seconds for consistency across metrics.
	return prometheus.ObserverFunc(func(v float64) {
		metric.Observe(v / 1e6)
	})
}
func (f *workqueueCustomMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return clientGoWorkqueueUnfinishedWorkSecondsMetricVec.WithLabelValues(name)
}
func (f *workqueueCustomMetricsProvider) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return clientGoWorkqueueLongestRunningProcessorMetricVec.WithLabelValues(name)
}
func (workqueueCustomMetricsProvider) NewRetriesMetric(name string) workqueue.CounterMetric {
	return clientGoWorkqueueRetrysMetricVec.WithLabelValues(name)
}