// Copyright 2021 The PipeCD Authors.
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

package migration

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/pipe-cd/pipe/pkg/datastore"
	"github.com/pipe-cd/pipe/pkg/model"
)

type DataTransfer interface {
	TransferMulti(ctx context.Context, kinds []string) error
}

type dataTransfer struct {
	source      datastore.DataStore
	destination datastore.DataStore
}

func NewDataTransfer(src, dest datastore.DataStore) DataTransfer {
	return &dataTransfer{
		source:      src,
		destination: dest,
	}
}

func transferOne(ctx context.Context, source, destination datastore.DataStore, kind string) error {
	it, err := source.Find(ctx, kind, datastore.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get data of kind %s from upstream datastore: %w", kind, err)
	}

	for {
		data, err := makeModelObject(kind)
		if err != nil {
			return err
		}

		err = it.Next(data)
		if err == datastore.ErrIteratorDone {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to get data of kind %s from datastore: %w", kind, err)
		}

		err = destination.Create(ctx, kind, data.GetId(), data)
		// Ignore ErrAlreadyExists to enable rerun from failed.
		if err == datastore.ErrAlreadyExists {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to insert data of kind %s (id: %s) to new datastore: %w", kind, data.GetId(), err)
		}
	}

	return nil
}

func (d *dataTransfer) TransferMulti(ctx context.Context, kinds []string) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, kind := range kinds {
		eg.Go(func() error {
			return transferOne(ctx, d.source, d.destination, kind)
		})
	}
	return eg.Wait()
}

type modelData interface {
	GetId() string
}

func makeModelObject(kind string) (modelData, error) {
	switch kind {
	case "Project":
		return &model.Project{}, nil
	case "Application":
		return &model.Application{}, nil
	case "Command":
		return &model.Command{}, nil
	case "Deployment":
		return &model.Deployment{}, nil
	case "Environment":
		return &model.Environment{}, nil
	case "Piped":
		return &model.Piped{}, nil
	case "APIKey":
		return &model.APIKey{}, nil
	case "Event":
		return &model.Event{}, nil
	default:
		return nil, fmt.Errorf("unsupported kind %s", kind)
	}
}
