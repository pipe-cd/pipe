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

package cloudrun

import (
	"context"
	"path/filepath"

	provider "github.com/pipe-cd/pipe/pkg/app/piped/cloudprovider/cloudrun"
	"github.com/pipe-cd/pipe/pkg/app/piped/executor"
	"github.com/pipe-cd/pipe/pkg/config"
	"github.com/pipe-cd/pipe/pkg/model"
)

type Executor struct {
	executor.Input

	config *config.CloudRunDeploymentSpec
}

type registerer interface {
	Register(stage model.Stage, f executor.Factory) error
	RegisterRollback(kind model.ApplicationKind, f executor.Factory) error
}

func Register(r registerer) {
	f := func(in executor.Input) executor.Executor {
		return &Executor{
			Input: in,
		}
	}

	r.Register(model.StageCloudRunSync, f)
	r.Register(model.StageCloudRunCanaryRollout, f)
	r.Register(model.StageCloudRunTrafficRouting, f)

	r.RegisterRollback(model.ApplicationKind_CLOUDRUN, f)
}

func (e *Executor) Execute(sig executor.StopSignal) model.StageStatus {
	e.config = e.DeploymentConfig.CloudRunDeploymentSpec
	if e.config == nil {
		e.LogPersister.Error("Malformed deployment configuration: missing CloudRunDeploymentSpec")
		return model.StageStatus_STAGE_FAILURE
	}

	var (
		ctx            = sig.Context()
		originalStatus = e.Stage.Status
		status         model.StageStatus
	)

	switch model.Stage(e.Stage.Name) {
	case model.StageCloudRunSync:
		status = e.ensureSync(ctx)

	case model.StageCloudRunCanaryRollout:
		status = e.ensureCanaryRollout(ctx)

	case model.StageCloudRunTrafficRouting:
		status = e.ensureTrafficRouting(ctx)

	case model.StageRollback:
		status = e.ensureRollback(ctx)

	default:
		e.LogPersister.Errorf("Unsupported stage %s for cloudrun application", e.Stage.Name)
		return model.StageStatus_STAGE_FAILURE
	}

	return executor.DetermineStageStatus(sig.Signal(), originalStatus, status)
}

func (e *Executor) ensureSync(ctx context.Context) model.StageStatus {
	var (
		commit = e.Deployment.Trigger.Commit.Hash
		appDir = filepath.Join(e.RepoDir, e.Deployment.GitPath.Path)
		p      = provider.NewProvider(appDir, e.config.Input, e.Logger)
	)

	e.LogPersister.Infof("Loading service manifest at the triggered commit %s", commit)
	sm, err := p.LoadServiceManifest()
	if err != nil {
		e.LogPersister.Errorf("Failed to load service manifest file (%v)", err)
		return model.StageStatus_STAGE_FAILURE
	}
	e.LogPersister.Info("Successfully loaded the service manifest")

	e.LogPersister.Info("Generate a service manifest that configures all traffic to the revision specified at the triggered commit")
	revision, err := provider.DecideRevisionName(sm, commit)
	if err != nil {
		e.LogPersister.Errorf("Unable to decide revision name for the commit %s (%v)", commit, err)
		return model.StageStatus_STAGE_FAILURE
	}

	if err := sm.SetRevision(revision); err != nil {
		e.LogPersister.Errorf("Unable to set revision name to service manifest (%v)", err)
		return model.StageStatus_STAGE_FAILURE
	}

	if err := sm.UpdateAllTraffic(revision); err != nil {
		e.LogPersister.Errorf("Unable to configure all traffic to revision %s (%v)", revision, err)
		return model.StageStatus_STAGE_FAILURE
	}
	e.LogPersister.Info("Successfully generated the appropriate service manifest")

	e.LogPersister.Info("Start applying the service manifest")
	if err := p.Apply(ctx, sm); err != nil {
		e.LogPersister.Errorf("Failed to apply the service manifest (%v)", err)
		return model.StageStatus_STAGE_FAILURE
	}
	e.LogPersister.Info("Successfully applied the service manifest")

	return model.StageStatus_STAGE_SUCCESS
}

func (e *Executor) ensureCanaryRollout(ctx context.Context) model.StageStatus {
	return model.StageStatus_STAGE_SUCCESS
}

func (e *Executor) ensureTrafficRouting(ctx context.Context) model.StageStatus {
	return model.StageStatus_STAGE_SUCCESS
}

func (e *Executor) ensureRollback(ctx context.Context) model.StageStatus {
	return model.StageStatus_STAGE_SUCCESS
}
