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

package model

import (
	"google.golang.org/protobuf/proto"
)

var notCompletedDeploymentStatuses = []DeploymentStatus{
	DeploymentStatus_DEPLOYMENT_PENDING,
	DeploymentStatus_DEPLOYMENT_PLANNED,
	DeploymentStatus_DEPLOYMENT_RUNNING,
	DeploymentStatus_DEPLOYMENT_ROLLING_BACK,
}

// IsCompletedDeployment checks whether the deployment is at a completion state.
func IsCompletedDeployment(status DeploymentStatus) bool {
	if status.String() == "" {
		return false
	}

	switch status {
	case DeploymentStatus_DEPLOYMENT_SUCCESS:
		return true
	case DeploymentStatus_DEPLOYMENT_FAILURE:
		return true
	case DeploymentStatus_DEPLOYMENT_CANCELLED:
		return true
	}
	return false
}

// IsCompletedStage checks whether the stage is at a completion state.
func IsCompletedStage(status StageStatus) bool {
	if status.String() == "" {
		return false
	}

	switch status {
	case StageStatus_STAGE_SUCCESS:
		return true
	case StageStatus_STAGE_FAILURE:
		return true
	case StageStatus_STAGE_CANCELLED:
		return true
	}
	return false
}

// CanUpdateDeploymentStatus checks whether the deployment can transit to the given status.
func CanUpdateDeploymentStatus(cur, next DeploymentStatus) bool {
	switch next {
	case DeploymentStatus_DEPLOYMENT_PENDING:
		return cur <= DeploymentStatus_DEPLOYMENT_PENDING
	case DeploymentStatus_DEPLOYMENT_PLANNED:
		return cur <= DeploymentStatus_DEPLOYMENT_PLANNED
	case DeploymentStatus_DEPLOYMENT_RUNNING:
		return cur <= DeploymentStatus_DEPLOYMENT_RUNNING
	case DeploymentStatus_DEPLOYMENT_ROLLING_BACK:
		return cur <= DeploymentStatus_DEPLOYMENT_ROLLING_BACK
	case DeploymentStatus_DEPLOYMENT_SUCCESS:
		return cur <= DeploymentStatus_DEPLOYMENT_ROLLING_BACK
	case DeploymentStatus_DEPLOYMENT_FAILURE:
		return cur <= DeploymentStatus_DEPLOYMENT_ROLLING_BACK
	case DeploymentStatus_DEPLOYMENT_CANCELLED:
		return cur <= DeploymentStatus_DEPLOYMENT_ROLLING_BACK
	}
	return false
}

// GetNotCompletedDeploymentStatuses returns DeploymentStatus slice that is not completed.
func GetNotCompletedDeploymentStatuses() []DeploymentStatus {
	return notCompletedDeploymentStatuses
}

// CanUpdateStageStatus checks whether the stage can transit to the given status.
func CanUpdateStageStatus(cur, next StageStatus) bool {
	switch next {
	case StageStatus_STAGE_NOT_STARTED_YET:
		return cur <= StageStatus_STAGE_NOT_STARTED_YET
	case StageStatus_STAGE_RUNNING:
		return cur <= StageStatus_STAGE_RUNNING
	case StageStatus_STAGE_SUCCESS:
		return cur <= StageStatus_STAGE_RUNNING
	case StageStatus_STAGE_FAILURE:
		return cur <= StageStatus_STAGE_RUNNING
	case StageStatus_STAGE_CANCELLED:
		return cur <= StageStatus_STAGE_RUNNING
	}
	return false
}

// StageStatusMap returns the map from id to status of all stages.
func (d *Deployment) StageStatusMap() map[string]StageStatus {
	statuses := make(map[string]StageStatus, len(d.Stages))
	for _, s := range d.Stages {
		statuses[s.Id] = s.Status
	}
	return statuses
}

// CommitHash returns the hash value of trigger commit.
func (d *Deployment) CommitHash() string {
	if c := d.Trigger.Commit; c != nil {
		return c.Hash
	}
	return ""
}

// ShortCommitHash returns the first seven characters of the hash.
func (d *Deployment) ShortCommitHash() string {
	h := d.CommitHash()
	if len(h) < 8 {
		return h
	}
	return h[:8]
}

// Clone returns a deep copy of the deployment.
func (d *Deployment) Clone() *Deployment {
	msg := proto.Clone(d)
	return msg.(*Deployment)
}

// TriggerBefore reports whether the deployment is triggered before the given one.
func (d *Deployment) TriggerBefore(other *Deployment) bool {
	if d.Trigger.Commit.CreatedAt > other.Trigger.Commit.CreatedAt {
		return false
	}
	if d.Trigger.Commit.CreatedAt == other.Trigger.Commit.CreatedAt && d.Trigger.Timestamp > other.Trigger.Timestamp {
		return false
	}
	return true

}

// CloudProviderType determines the cloud provider type from application kind.
func (d *Deployment) CloudProviderType() CloudProviderType {
	switch d.Kind {
	case ApplicationKind_KUBERNETES:
		return CloudProviderKubernetes
	case ApplicationKind_TERRAFORM:
		return CloudProviderTerraform
	case ApplicationKind_CLOUDRUN:
		return CloudProviderCloudRun
	case ApplicationKind_LAMBDA:
		return CloudProviderLambda
	default:
		return CloudProviderType(d.Kind.String())
	}
}

// FindRollbackStage finds the rollback stage in stage list.
func (d *Deployment) FindRollbackStage() (*PipelineStage, bool) {
	for i := len(d.Stages) - 1; i >= 0; i-- {
		if d.Stages[i].Name == StageRollback.String() {
			return d.Stages[i], true
		}
	}
	return nil, false
}
