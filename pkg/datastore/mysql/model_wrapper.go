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

package mysql

import (
	"encoding/json"
	"fmt"

	"github.com/pipe-cd/pipe/pkg/model"
)

func wrapModel(entity interface{}) (interface{}, error) {
	switch e := entity.(type) {
	case *model.Project:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &project{
			Project: *e,
			Extra:   e.GetId(),
		}, nil
	case *model.Application:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &application{
			Application: *e,
			Extra:       e.GetName(),
		}, nil
	case *model.Command:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &command{
			Command: *e,
			Extra:   e.GetId(),
		}, nil
	case *model.Deployment:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &deployment{
			Deployment: *e,
			Extra:      e.GetId(),
		}, nil
	case *model.Environment:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &environment{
			Environment: *e,
			Extra:       e.GetName(),
		}, nil
	case *model.Piped:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &piped{
			Piped: *e,
			Extra: e.GetName(),
		}, nil
	case *model.APIKey:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &apiKey{
			APIKey: *e,
			Extra:  e.GetName(),
		}, nil
	case *model.Event:
		if e == nil {
			return nil, fmt.Errorf("nil entity given")
		}
		return &event{
			Event: *e,
			Extra: e.GetName(),
		}, nil
	default:
		return nil, fmt.Errorf("%T is not supported", e)
	}
}

func encodeJSONValue(entity interface{}) (string, error) {
	wrapper, err := wrapModel(entity)
	if err != nil {
		return "", err
	}
	encodedEntity, err := json.Marshal(wrapper)
	if err != nil {
		return "", err
	}
	return string(encodedEntity), nil
}

func decodeJSONValue(val string, target interface{}) error {
	return json.Unmarshal([]byte(val), target)
}

type project struct {
	model.Project `json:",inline"`
	Extra         string `json:"extra"`
}

type application struct {
	model.Application `json:",inline"`
	Extra             string `json:"extra"`
}

type command struct {
	model.Command `json:",inline"`
	Extra         string `json:"extra"`
}

type deployment struct {
	model.Deployment `json:",inline"`
	Extra            string `json:"extra"`
}

type environment struct {
	model.Environment `json:",inline"`
	Extra             string `json:"extra"`
}

type piped struct {
	model.Piped `json:",inline"`
	Extra       string `json:"extra"`
}

type apiKey struct {
	model.APIKey `json:",inline"`
	Extra        string `json:"extra"`
}

type event struct {
	model.Event `json:",inline"`
	Extra       string `json:"extra"`
}
