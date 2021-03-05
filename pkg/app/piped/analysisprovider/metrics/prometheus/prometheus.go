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

package prometheus

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/pipe-cd/pipe/pkg/config"
)

const (
	ProviderType   = "Prometheus"
	defaultTimeout = 30 * time.Second
)

// Provider is a client for prometheus.
type Provider struct {
	api v1.API
	//username string
	//password string

	timeout time.Duration
	logger  *zap.Logger
}

func NewProvider(address string, timeout time.Duration, logger *zap.Logger) (*Provider, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &Provider{
		api:     v1.NewAPI(client),
		timeout: timeout,
		logger:  logger.With(zap.String("analysis-provider", ProviderType)),
	}, nil
}

func (p *Provider) Type() string {
	return ProviderType
}

func (p *Provider) RunQuery(ctx context.Context, query string, expected config.AnalysisExpected) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	p.logger.Info("run query", zap.String("query", query))
	// TODO: Use HTTP Basic Authentication with the username and password when needed.
	response, warnings, err := p.api.Query(ctx, query, time.Now())
	if errors.Is(err, context.DeadlineExceeded) {
		// Treat as a success because query duration is already ended.
		return true, nil
	}
	if err != nil {
		return false, err
	}
	for _, w := range warnings {
		p.logger.Warn("non critical error occurred", zap.String("warning", w))
	}
	return p.evaluate(expected, response)
}

func (p *Provider) evaluate(expected config.AnalysisExpected, response model.Value) (bool, error) {
	switch value := response.(type) {
	case *model.Scalar:
		result := float64(value.Value)
		if math.IsNaN(result) {
			return false, fmt.Errorf("the result %v is not a number", result)
		}
		return p.inRange(expected, result)
	case model.Vector:
		lv := len(value)
		if lv == 0 {
			return false, fmt.Errorf("zero value returned")
		}
		results := make([]float64, 0, lv)
		for _, s := range value {
			if s == nil {
				continue
			}
			result := float64(s.Value)
			if math.IsNaN(result) {
				return false, fmt.Errorf("the result %v is not a number", result)
			}
			results = append(results, result)
		}
		p.logger.Info("vector results found", zap.Float64s("results", results))
		// TODO: Consider the case of multiple results.
		return p.inRange(expected, results[0])
	default:
		return false, fmt.Errorf("unsupported prometheus metrics type")
	}
}

func (p *Provider) inRange(expected config.AnalysisExpected, value float64) (bool, error) {
	if expected.Min == nil && expected.Max == nil {
		return false, fmt.Errorf("expected range is undefined")
	}
	if min := expected.Min; min != nil && *min > value {
		p.logger.Info("failure because the query response was below expected minimum", zap.Float64("response", value), zap.Float64("expected-min", *min))
		return false, nil
	}
	if max := expected.Max; max != nil && *max < value {
		p.logger.Info("failure because the query response exceeded expected maximum", zap.Float64("response", value), zap.Float64("expected-max", *max))
		return false, nil
	}
	return true, nil
}
