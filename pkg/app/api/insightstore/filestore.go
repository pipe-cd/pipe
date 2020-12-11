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
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pipe-cd/pipe/pkg/filestore"
	"github.com/pipe-cd/pipe/pkg/model"
)

var (
	eol = []byte("EOL")
)

type insightFileStore struct {
	filestore filestore.Store
}

func (f *insightFileStore) List(
	ctx context.Context,
	projectID string,
	appID string,
	metricsKind model.InsightMetricsKind,
	step model.InsightStep,
	from time.Time,
	dataPointCount int) ([]*model.InsightDataPoint, error) {
	from = formatFrom(from, step)

	paths := insightFilePaths(projectID, appID, from, dataPointCount, metricsKind, step)

	var idps []*model.InsightDataPoint
	for _, p := range paths {
		obj, err := f.filestore.GetObject(ctx, p)
		if err != nil {
			return nil, err
		}
		idp, err := f.getInsightDataPoints(obj, from, dataPointCount, step, metricsKind)
		if err != nil {
			return nil, err
		}

		idps = append(idps, idp...)
	}

	return idps, nil
}

func (f *insightFileStore) getInsightDataPoints(obj filestore.Object, from time.Time, dataPointCount int, step model.InsightStep, kind model.InsightMetricsKind) ([]*model.InsightDataPoint, error) {
	var c commonReport
	if err := json.Unmarshal(obj.Content, &c); err != nil {
		return nil, err
	}

	var getKey func(t time.Time) string
	var nextTargetDate func(t time.Time) time.Time
	var targetJSON []byte
	switch step {
	case model.InsightStep_YEARLY:
		targetJSON = c.Datapoints.Yearly
		getKey = func(t time.Time) string {
			return strconv.Itoa(t.Year())
		}
		nextTargetDate = func(t time.Time) time.Time {
			return t.AddDate(1, 0, 0)
		}
	case model.InsightStep_MONTHLY:
		targetJSON = c.Datapoints.Monthly
		getKey = func(t time.Time) string {
			return t.Format("2006-01")
		}
		nextTargetDate = func(t time.Time) time.Time {
			return t.AddDate(0, 1, 0)
		}
	case model.InsightStep_WEEKLY:
		targetJSON = c.Datapoints.Weekly
		getKey = func(t time.Time) string {
			// This day must be a Sunday, otherwise it will fail to get the value from the map.
			return t.Format("2006-01-02")
		}
		nextTargetDate = func(t time.Time) time.Time {
			return t.AddDate(0, 0, 7)
		}
	case model.InsightStep_DAILY:
		targetJSON = c.Datapoints.Daily
		getKey = func(t time.Time) string {
			return t.Format("2006-01-02")
		}
		nextTargetDate = func(t time.Time) time.Time {
			return t.AddDate(0, 0, 1)
		}
	}

	var points map[string]datapoint
	switch kind {
	case model.InsightMetricsKind_DEPLOYMENT_FREQUENCY:
		var df map[string]deployFrequency
		err := json.Unmarshal(targetJSON, &df)
		if err != nil {
			return nil, err
		}
		points, err = toDatapoint(df)
		if err != nil {
			return nil, err
		}
	case model.InsightMetricsKind_CHANGE_FAILURE_RATE:
		var cfr map[string]changeFailureRate
		err := json.Unmarshal(targetJSON, &cfr)
		if err != nil {
			return nil, err
		}
		points, err = toDatapoint(cfr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unimpremented insight kind: %s", kind)
	}

	idps := make([]*model.InsightDataPoint, dataPointCount)
	targetDate := from
	for i := 0; i < dataPointCount; i++ {
		key := getKey(targetDate)
		d, ok := points[key]
		if !ok {
			return nil, fmt.Errorf("datapoints not found, key: %s", key)
		}

		idps[i] = &model.InsightDataPoint{
			Value:     d.Value(),
			Timestamp: targetDate.Unix(),
		}

		targetDate = nextTargetDate(targetDate)
	}

	return idps, nil
}

func formatFrom(from time.Time, step model.InsightStep) time.Time {
	var formattedTime time.Time
	switch step {
	case model.InsightStep_DAILY:
		formattedTime = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	case model.InsightStep_WEEKLY:
		// Sunday in the week of rangeFrom
		sunday := from.AddDate(0, 0, -int(from.Weekday()))
		formattedTime = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 0, 0, 0, 0, time.UTC)
	case model.InsightStep_MONTHLY:
		formattedTime = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	case model.InsightStep_YEARLY:
		formattedTime = time.Date(from.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return formattedTime
}

// insightFilePaths return an insight file paths according to the following format
//
// insights
//  ├─ project-id
//    ├─ deployment-frequency
//        ├─ project  # aggregated from all applications
//            ├─ years.json
//            ├─ 2020-01.json
//            ├─ 2020-02.json
//            ...
//        ├─ app-id
//            ├─ years.json
//            ├─ 2020-01.json
//            ├─ 2020-02.json
//            ...
func insightFilePaths(projectID string, appID string, from time.Time, dataPointCount int, metricsKind model.InsightMetricsKind, step model.InsightStep) []string {
	if appID == "" {
		appID = "project"
	}
	metricsKindKebab := getKebabCaseMetricsKind(metricsKind)
	switch step {
	case model.InsightStep_YEARLY:
		return []string{fmt.Sprintf("insights/%s/%s/%s/years.json", projectID, metricsKindKebab, appID)}
	default:
		months := getPointsMonths(from, dataPointCount, step)
		var paths []string
		for _, m := range months {
			path := fmt.Sprintf("insights/%s/%s/%s/%s.json", projectID, metricsKindKebab, appID, m)
			paths = append(paths, path)
		}
		return paths
	}
}

// getPointsMonths return months between two dates.
func getPointsMonths(date time.Time, count int, step model.InsightStep) []string {
	var to time.Time

	switch step {
	case model.InsightStep_YEARLY:
		to = date.AddDate(count-1, 0, 0)
	case model.InsightStep_MONTHLY:
		to = date.AddDate(0, count-1, 0)
	case model.InsightStep_WEEKLY:
		to = date.AddDate(0, 0, (count-1)*7)
	case model.InsightStep_DAILY:
		to = date.AddDate(0, 0, count-1)
	}

	var months []string
	y1, m1, _ := to.Date()
	for {
		// 2015-05-05 08:05:15.828452891 +0900 UST → 2015-05
		months = append(months, date.Format("2006-01"))
		y2, m2, _ := date.Date()
		if y1 == y2 && m1 == m2 {
			return months
		}

		date = date.AddDate(0, 1, 0)
	}
}

func getKebabCaseMetricsKind(kind model.InsightMetricsKind) string {
	var kebabKind string
	switch kind {
	case model.InsightMetricsKind_DEPLOYMENT_FREQUENCY:
		kebabKind = "deployment_frequency"
	case model.InsightMetricsKind_CHANGE_FAILURE_RATE:
		kebabKind = "change_failure_rate"
	case model.InsightMetricsKind_MTTR:
		kebabKind = "mean_time_to_restore"
	case model.InsightMetricsKind_LEAD_TIME:
		kebabKind = "lead_time"
	}
	return kebabKind
}
