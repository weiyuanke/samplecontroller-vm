>Extend sample-controller to expose the following metrics so that prometheus server can collect metrics
>```
> workqueue related metrics 
> reflector related metrics
> k8s client related metrics
>```
>

package "workqueue" provide an extension point 

```go
// SetProvider sets the metrics provider for all subsequently created work
// queues. Only the first call has an effect.
func SetProvider(metricsProvider MetricsProvider) {
	globalMetricsFactory.setProvider(metricsProvider)
}
```

and provides the following metrics of the workqueue

```go
// MetricsProvider generates various metrics used by the queue.
type MetricsProvider interface {
    // current depth of a workqueue
	NewDepthMetric(name string) GaugeMetric
    // total number of adds handled by a workqueue
	NewAddsMetric(name string) CounterMetric
    // how long an item stays in a workqueue
	NewLatencyMetric(name string) HistogramMetric
	NewWorkDurationMetric(name string) HistogramMetric
	NewUnfinishedWorkSecondsMetric(name string) SettableGaugeMetric
	NewLongestRunningProcessorSecondsMetric(name string) SettableGaugeMetric
	NewRetriesMetric(name string) CounterMetric
}
```

we can build a MetricsProvider based on prometheus -- CustomMetricsProvider.go

```go
	// set Custom Metrics provider
	customMetricsProvider := &workqueueCustomMetricsProvider{}
	customMetricsProvider.Register(prometheus.DefaultRegisterer)
```

the whole link is like:
queue.add -> q.metrics.add -> defaultQueueMetrics -> CustomMetricsProvider

