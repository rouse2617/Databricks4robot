// Package cloudmonitoring writes custom metrics directly to GCP Cloud
// Monitoring. Used as a lightweight alternative to GMP (Managed Prometheus)
// for the few dispatcher gauges the reconciler emits — no collector, no
// additional GCP API enablement needed beyond monitoring.googleapis.com.
//
// The service account running the backend must have the
// roles/monitoring.metricWriter IAM binding on the project (see SETUP.md /
// current-work.md for the dev SA). Without it, WriteInt64Metric returns a
// permission-denied error but the reconciler logs and continues.
package cloudmonitoring

import (
	"context"
	"fmt"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const metricPrefix = "custom.googleapis.com/"

// Writer writes custom metrics to GCP Cloud Monitoring via the WriteTimeSeries
// API. Safe for concurrent use (the underlying client is goroutine-safe).
type Writer struct {
	client    *monitoring.MetricClient
	projectID string
}

// NewWriter creates a monitoring client using Application Default Credentials
// (the same ADC that PubSub, Storage, and BigQuery clients use in this repo).
// projectID is the GCP project to write metrics into (e.g. "green-valley-442103").
func NewWriter(ctx context.Context, projectID string) (*Writer, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloudmonitoring: new client: %w", err)
	}
	return &Writer{client: client, projectID: projectID}, nil
}

// WriteInt64Metric writes a single int64 gauge sample to Cloud Monitoring.
// metricType is the suffix after custom.googleapis.com/, e.g. "dispatcher/stale_items".
// The metric descriptor is auto-created on first write.
func (w *Writer) WriteInt64Metric(ctx context.Context, metricType string, value int64) error {
	if w == nil {
		return nil
	}
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: fmt.Sprintf("projects/%s", w.projectID),
		TimeSeries: []*monitoringpb.TimeSeries{{
			Metric: &metricpb.Metric{
				Type: metricPrefix + metricType,
			},
			Resource: &monitoredrespb.MonitoredResource{
				Type: "global",
			},
			Points: []*monitoringpb.Point{{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.New(time.Now()),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_Int64Value{Int64Value: value},
				},
			}},
		}},
	}
	return w.client.CreateTimeSeries(ctx, req)
}

// Close tears down the underlying gRPC connection.
func (w *Writer) Close() error {
	if w == nil || w.client == nil {
		return nil
	}
	return w.client.Close()
}
