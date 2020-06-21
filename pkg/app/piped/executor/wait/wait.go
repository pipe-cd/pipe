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

package wait

import (
	"fmt"
	"time"

	"github.com/pipe-cd/pipe/pkg/app/piped/executor"
	"github.com/pipe-cd/pipe/pkg/model"
)

var defaultDuration = time.Minute

type Executor struct {
	executor.Input
}

type registerer interface {
	Register(stage model.Stage, f executor.Factory) error
}

// Register registers this executor factory into a given registerer.
func Register(r registerer) {
	f := func(in executor.Input) executor.Executor {
		return &Executor{
			Input: in,
		}
	}
	r.Register(model.StageWait, f)
}

// Execute starts waiting for the specified duration.
// TODO: Persist and load the start time to deal with restart case.
func (e *Executor) Execute(sig executor.StopSignal) model.StageStatus {
	var (
		originalStatus = e.Stage.Status
		duration       = defaultDuration
		timer          = time.NewTimer(duration)
	)
	defer timer.Stop()

	if opts := e.StageConfig.WaitStageOptions; opts != nil {
		if opts.Duration > 0 {
			duration = opts.Duration.Duration()
		}
	}

	e.LogPersister.AppendInfo(fmt.Sprintf("Waiting for %v...", duration))
	select {
	case <-timer.C:
		break
	case s := <-sig.Ch():
		switch s {
		case executor.StopSignalCancel:
			return model.StageStatus_STAGE_CANCELLED
		case executor.StopSignalTerminate:
			return originalStatus
		default:
			return model.StageStatus_STAGE_FAILURE
		}
	}

	e.LogPersister.AppendInfo(fmt.Sprintf("Waited for %v", duration))
	return model.StageStatus_STAGE_SUCCESS
}
