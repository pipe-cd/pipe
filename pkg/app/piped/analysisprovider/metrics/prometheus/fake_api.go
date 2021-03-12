package prometheus

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type fakeAPI struct {
	value    model.Value
	err      error
	warnings v1.Warnings
}

func (m fakeAPI) Query(_ context.Context, _ string, _ time.Time) (model.Value, v1.Warnings, error) {
	if m.err != nil {
		return nil, m.warnings, m.err
	}
	return m.value, m.warnings, nil
}

func (m fakeAPI) QueryRange(_ context.Context, _ string, _ v1.Range) (model.Value, v1.Warnings, error) {
	if m.err != nil {
		return nil, m.warnings, m.err
	}
	return m.value, m.warnings, nil
}

// Below methods are required to meet the interface.

func (m fakeAPI) Metadata(ctx context.Context, metric string, limit string) (map[string][]v1.Metadata, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) CleanTombstones(ctx context.Context) error {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) LabelNames(ctx context.Context) ([]string, v1.Warnings, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) LabelValues(ctx context.Context, label string) (model.LabelValues, v1.Warnings, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, v1.Warnings, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Targets(ctx context.Context) (v1.TargetsResult, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Alerts(ctx context.Context) (v1.AlertsResult, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) AlertManagers(ctx context.Context) (v1.AlertManagersResult, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Config(ctx context.Context) (v1.ConfigResult, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Flags(ctx context.Context) (v1.FlagsResult, error) {
	panic("this method doesn't expect to be called")
}

func (m fakeAPI) Snapshot(ctx context.Context, skipHead bool) (v1.SnapshotResult, error) {
	panic("Not used")
}

func (m fakeAPI) Rules(ctx context.Context) (v1.RulesResult, error) {
	panic("Not used")
}

func (m fakeAPI) TargetsMetadata(ctx context.Context, matchTarget string, metric string, limit string) ([]v1.MetricMetadata, error) {
	panic("Not used")
}
