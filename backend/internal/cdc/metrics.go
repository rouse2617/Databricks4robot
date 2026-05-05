package cdc

import "github.com/prometheus/client_golang/prometheus"

// CDC Prometheus metrics. Exported so other files in the package can update them.
var (
	// CDCConsumerLag tracks the number of events behind the head of the topic,
	// labelled by consumer_kind.
	CDCConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cdc_consumer_lag_events",
			Help: "Number of events the consumer is behind the topic head, by consumer_kind",
		},
		[]string{"consumer_kind"},
	)

	// CDCBatchProcessedTotal counts batches processed, labelled by consumer_kind.
	CDCBatchProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cdc_batch_processed_total",
			Help: "Total number of CDC events processed per batch, by consumer_kind",
		},
		[]string{"consumer_kind"},
	)

	// CDCDecodeErrorsTotal counts Debezium message decode failures.
	CDCDecodeErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cdc_decode_errors_total",
			Help: "Total number of Debezium message decode failures",
		},
	)

	// CDCESRebuildDurationMs records the duration of ES rebuild operations in milliseconds.
	CDCESRebuildDurationMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "cdc_es_rebuild_duration_ms",
			Help:    "Duration of Elasticsearch rebuild operations in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
	)
)

func init() {
	prometheus.MustRegister(
		CDCConsumerLag,
		CDCBatchProcessedTotal,
		CDCDecodeErrorsTotal,
		CDCESRebuildDurationMs,
	)
}
