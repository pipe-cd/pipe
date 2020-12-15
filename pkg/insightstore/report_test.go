// Copyright 2020 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package insightstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pipe-cd/pipe/pkg/model"
)

func Test_convertToInsightDataPoints(t *testing.T) {
	type args struct {
		report         Report
		from           time.Time
		dataPointCount int
		step           model.InsightStep
	}
	tests := []struct {
		name    string
		args    args
		want    []*model.InsightDataPoint
		wantErr bool
	}{
		{
			name: "success with yearly",
			args: args{
				report: func() Report {
					path := makeYearsFilePath("projectID", model.InsightMetricsKind_DEPLOYMENT_FREQUENCY, "appID")
					expected := DeployFrequencyReport{
						AccumulatedTo: 1609459200,
						Datapoints: DeployFrequencyDataPoint{
							Yearly: map[string]DeployFrequency{
								"2020": {DeployCount: 1000},
								"2021": {DeployCount: 3000},
							},
						},
						FilePath: path,
					}
					report, _ := toReport(&expected)
					return report
				}(),
				from:           time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				dataPointCount: 2,
				step:           model.InsightStep_YEARLY,
			},
			want: []*model.InsightDataPoint{
				{
					Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     1000,
				},
				{
					Timestamp: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     3000,
				},
			},
		},
		{
			name: "success with monthly",
			args: args{
				report: func() Report {
					path := makeYearsFilePath("projectID", model.InsightMetricsKind_DEPLOYMENT_FREQUENCY, "appID")
					expected := DeployFrequencyReport{
						AccumulatedTo: 1609459200,
						Datapoints: DeployFrequencyDataPoint{
							Monthly: map[string]DeployFrequency{
								"2020-01": {DeployCount: 1000},
							},
						},
						FilePath: path,
					}
					report, _ := toReport(&expected)
					return report
				}(),
				from:           time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				dataPointCount: 1,
				step:           model.InsightStep_MONTHLY,
			},
			want: []*model.InsightDataPoint{
				{
					Timestamp: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     1000,
				},
			},
		},
		{
			name: "success with weekly",
			args: args{
				report: func() Report {
					path := makeYearsFilePath("projectID", model.InsightMetricsKind_DEPLOYMENT_FREQUENCY, "appID")
					expected := DeployFrequencyReport{
						AccumulatedTo: 1609459200,
						Datapoints: DeployFrequencyDataPoint{
							Weekly: map[string]DeployFrequency{
								"2021-01-03": {DeployCount: 1000},
								"2021-01-10": {DeployCount: 3000},
							},
						},
						FilePath: path,
					}
					report, _ := toReport(&expected)
					return report
				}(),
				from:           time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				dataPointCount: 2,
				step:           model.InsightStep_WEEKLY,
			},
			want: []*model.InsightDataPoint{
				{
					Timestamp: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     1000,
				},
				{
					Timestamp: time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     3000,
				},
			},
		},
		{
			name: "success with daily",
			args: args{
				report: func() Report {
					path := makeYearsFilePath("projectID", model.InsightMetricsKind_DEPLOYMENT_FREQUENCY, "appID")
					expected := DeployFrequencyReport{
						AccumulatedTo: 1609459200,
						Datapoints: DeployFrequencyDataPoint{
							Daily: map[string]DeployFrequency{
								"2021-01-03": {DeployCount: 1000},
								"2021-01-04": {DeployCount: 3000},
							},
						},
						FilePath: path,
					}
					report, _ := toReport(&expected)
					return report
				}(),
				from:           time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				dataPointCount: 2,
				step:           model.InsightStep_DAILY,
			},
			want: []*model.InsightDataPoint{
				{
					Timestamp: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     1000,
				},
				{
					Timestamp: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC).Unix(),
					Value:     3000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertToInsightDataPoints(tt.args.report, tt.args.from, tt.args.dataPointCount, tt.args.step)
			if (err != nil) != tt.wantErr {
				if !tt.wantErr {
					assert.NoError(t, err)
					return
				}
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
