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

	// CDCKafkaRetriesTotal counts transient poll retries.
	CDCKafkaRetriesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cdc_kafka_retries_total",
			Help: "Total number of transient Kafka poll retries",
		},
	)

	// CDCUnknownTopicTotal counts batches dropped due to unknown topic routing.
	CDCUnknownTopicTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cdc_unknown_topic_total",
			Help: "Total number of CDC batches dropped because no topic handler was registered",
		},
	)

	// CDCHandlerFailuresTotal counts failures by processing stage.
	CDCHandlerFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cdc_handler_failures_total",
			Help: "Total number of CDC processing failures by stage",
		},
		[]string{"stage"},
	)

	// CDCDLQWriteFailuresTotal counts DLQ sink write failures.
	CDCDLQWriteFailuresTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cdc_dlq_write_failures_total",
			Help: "Total number of failures writing CDC failed records into DLQ sink",
		},
	)

	// CDCDLQWritesTotal counts successful DLQ writes.
	CDCDLQWritesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cdc_dlq_writes_total",
			Help: "Total number of CDC failed records written into DLQ sink",
		},
	)

	// CDCConsecutiveFailures tracks current consecutive failures in source loop.
	CDCConsecutiveFailures = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cdc_consecutive_failures",
			Help: "Current number of consecutive CDC record failures in source loop",
		},
	)
)

func init() {
	prometheus.MustRegister(
		CDCConsumerLag,
		CDCBatchProcessedTotal,
		CDCDecodeErrorsTotal,
		CDCESRebuildDurationMs,
		CDCKafkaRetriesTotal,
		CDCUnknownTopicTotal,
		CDCHandlerFailuresTotal,
		CDCDLQWriteFailuresTotal,
		CDCDLQWritesTotal,
		CDCConsecutiveFailures,
	)
}
