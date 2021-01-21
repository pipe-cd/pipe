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

package lambda

import (
	"context"
	"errors"
	"fmt"
	"time"

	provider "github.com/pipe-cd/pipe/pkg/app/piped/cloudprovider/lambda"
	"github.com/pipe-cd/pipe/pkg/app/piped/deploysource"
	"github.com/pipe-cd/pipe/pkg/app/piped/executor"
	"github.com/pipe-cd/pipe/pkg/backoff"
	"github.com/pipe-cd/pipe/pkg/config"
	"github.com/pipe-cd/pipe/pkg/model"
)

type registerer interface {
	Register(stage model.Stage, f executor.Factory) error
	RegisterRollback(kind model.ApplicationKind, f executor.Factory) error
}

func Register(r registerer) {
	f := func(in executor.Input) executor.Executor {
		return &deployExecutor{
			Input: in,
		}
	}
	r.Register(model.StageLambdaSync, f)
	r.Register(model.StageLambdaPromote, f)
	r.Register(model.StageLambdaCanaryRollout, f)

	r.RegisterRollback(model.ApplicationKind_LAMBDA, func(in executor.Input) executor.Executor {
		return &rollbackExecutor{
			Input: in,
		}
	})
}

func findCloudProvider(in *executor.Input) (name string, cfg *config.CloudProviderLambdaConfig, found bool) {
	name = in.Application.CloudProvider
	if name == "" {
		in.LogPersister.Errorf("Missing the CloudProvider name in the application configuration")
		return
	}

	cp, ok := in.PipedConfig.FindCloudProvider(name, model.CloudProviderLambda)
	if !ok {
		in.LogPersister.Errorf("The specified cloud provider %q was not found in piped configuration", name)
		return
	}

	cfg = cp.LambdaConfig
	found = true
	return
}

func loadFunctionManifest(in *executor.Input, functionManifestFile string, ds *deploysource.DeploySource) (provider.FunctionManifest, bool) {
	in.LogPersister.Infof("Loading service manifest at the %s commit (%s)", ds.RevisionName, ds.RevisionName)

	fm, err := provider.LoadFunctionManifest(ds.AppDir, functionManifestFile)
	if err != nil {
		in.LogPersister.Errorf("Failed to load lambda function manifest (%v)", err)
		return provider.FunctionManifest{}, false
	}

	in.LogPersister.Infof("Successfully loaded the lambda function manifest at the %s commit", ds.RevisionName)
	return fm, true
}

func sync(ctx context.Context, in *executor.Input, cloudProviderName string, cloudProviderCfg *config.CloudProviderLambdaConfig, fm provider.FunctionManifest) bool {
	in.LogPersister.Infof("Start applying the lambda function manifest")
	client, err := provider.DefaultRegistry().Client(cloudProviderName, cloudProviderCfg, in.Logger)
	if err != nil {
		in.LogPersister.Errorf("Unable to create Lambda client for the provider %s: %v", cloudProviderName, err)
		return false
	}

	// Build and publish new version of Lambda function.
	version, ok := build(ctx, in, client, fm)
	if !ok {
		in.LogPersister.Errorf("Failed to build new version for Lambda function %s", fm.Spec.Name)
		return false
	}

	trafficCfg, err := client.GetTrafficConfig(ctx, fm)
	// Store the current traffic config if existed for rollback if necessary.
	if trafficCfg != nil {
		originalTrafficCfg, ok := trafficCfg.Encode()
		if !ok {
			in.LogPersister.Errorf("Unable to store current traffic config for rollback: encode failed")
			return false
		}
		originalTrafficKeyName := fmt.Sprintf("%s-%s-original", fm.Spec.Name, in.Deployment.RunningCommitHash)
		if e := in.MetadataStore.Set(ctx, originalTrafficKeyName, originalTrafficCfg); e != nil {
			in.LogPersister.Errorf("Unable to store current traffic config for rollback: %v", e)
			return false
		}
	}
	// Create Alias on not yet existed.
	if errors.Is(err, provider.ErrNotFound) {
		if err := client.CreateTrafficConfig(ctx, fm, version); err != nil {
			in.LogPersister.Errorf("Failed to create traffic routing for Lambda function %s (version: %s): %v", fm.Spec.Name, version, err)
			return false
		}
		in.LogPersister.Infof("Successfully applied the lambda function manifest")
		return true
	}
	if err != nil {
		in.LogPersister.Errorf("Failed to prepare traffic routing for Lambda function %s: %v", fm.Spec.Name, err)
		return false
	}

	// Update 100% traffic to the new lambda version.
	if !configureTrafficRouting(trafficCfg, version, 100) {
		in.LogPersister.Errorf("Failed to prepare traffic routing for Lambda function %s", fm.Spec.Name)
		return false
	}

	if err = client.UpdateTrafficConfig(ctx, fm, trafficCfg); err != nil {
		in.LogPersister.Errorf("Failed to update traffic routing for Lambda function %s (version: %s): %v", fm.Spec.Name, version, err)
		return false
	}

	in.LogPersister.Infof("Successfully applied the manifest for Lambda function %s version (v%s)", fm.Spec.Name, version)
	return true
}

func rollout(ctx context.Context, in *executor.Input, cloudProviderName string, cloudProviderCfg *config.CloudProviderLambdaConfig, fm provider.FunctionManifest) bool {
	in.LogPersister.Infof("Start rolling out the lambda function: %s", fm.Spec.Name)
	client, err := provider.DefaultRegistry().Client(cloudProviderName, cloudProviderCfg, in.Logger)
	if err != nil {
		in.LogPersister.Errorf("Unable to create Lambda client for the provider %s: %v", cloudProviderName, err)
		return false
	}

	// Build and publish new version of Lambda function.
	version, ok := build(ctx, in, client, fm)
	if !ok {
		in.LogPersister.Errorf("Failed to build new version for Lambda function %s", fm.Spec.Name)
		return false
	}

	// Update rolled out version name to metadata store
	rolloutVersionKeyName := fmt.Sprintf("%s-rollout", fm.Spec.Name)
	if err := in.MetadataStore.Set(ctx, rolloutVersionKeyName, version); err != nil {
		in.LogPersister.Errorf("Failed to update latest version name to metadata store for Lambda function %s: %v", fm.Spec.Name, err)
		return false
	}

	// Store current traffic config for rollback if necessary.
	if trafficCfg, err := client.GetTrafficConfig(ctx, fm); err == nil {
		// Store the current traffic config.
		originalTrafficCfg, ok := trafficCfg.Encode()
		if !ok {
			in.LogPersister.Errorf("Unable to store current traffic config for rollback: encode failed")
			return false
		}
		originalTrafficKeyName := fmt.Sprintf("%s-%s-original", fm.Spec.Name, in.Deployment.RunningCommitHash)
		if e := in.MetadataStore.Set(ctx, originalTrafficKeyName, originalTrafficCfg); e != nil {
			in.LogPersister.Errorf("Unable to store current traffic config for rollback: %v", e)
			return false
		}
	}

	return true
}

func promote(ctx context.Context, in *executor.Input, cloudProviderName string, cloudProviderCfg *config.CloudProviderLambdaConfig, fm provider.FunctionManifest) bool {
	in.LogPersister.Infof("Start promote new version of the lambda function: %s", fm.Spec.Name)
	client, err := provider.DefaultRegistry().Client(cloudProviderName, cloudProviderCfg, in.Logger)
	if err != nil {
		in.LogPersister.Errorf("Unable to create Lambda client for the provider %s: %v", cloudProviderName, err)
		return false
	}

	rolloutVersionKeyName := fmt.Sprintf("%s-rollout", fm.Spec.Name)
	version, ok := in.MetadataStore.Get(rolloutVersionKeyName)
	if !ok {
		in.LogPersister.Errorf("Unable to prepare version to promote for Lambda function %s: Not found", fm.Spec.Name)
		return false
	}

	options := in.StageConfig.LambdaPromoteStageOptions
	if options == nil {
		in.LogPersister.Errorf("Malformed configuration for stage %s", in.Stage.Name)
		return false
	}

	trafficCfg, err := client.GetTrafficConfig(ctx, fm)
	// Create Alias on not yet existed.
	if errors.Is(err, provider.ErrNotFound) {
		if options.Percent != 100 {
			in.LogPersister.Errorf("Not previous version available to handle traffic, new version has to get 100 percent of traffic")
			return false
		}
		if err := client.CreateTrafficConfig(ctx, fm, version); err != nil {
			in.LogPersister.Errorf("Failed to create traffic routing for Lambda function %s (version: %s): %v", fm.Spec.Name, version, err)
			return false
		}
		in.LogPersister.Infof("Successfully route all traffic to the lambda function %s (version %s)", fm.Spec.Name, version)
		return true
	}
	if err != nil {
		in.LogPersister.Errorf("Failed to prepare traffic routing for Lambda function %s: %v", fm.Spec.Name, err)
		return false
	}

	// Update traffic to the new lambda version.
	if !configureTrafficRouting(trafficCfg, version, options.Percent) {
		in.LogPersister.Errorf("Failed to prepare traffic routing for Lambda function %s", fm.Spec.Name)
		return false
	}

	if err = client.UpdateTrafficConfig(ctx, fm, trafficCfg); err != nil {
		in.LogPersister.Errorf("Failed to update traffic routing for Lambda function %s (version: %s): %v", fm.Spec.Name, version, err)
		return false
	}

	in.LogPersister.Infof("Successfully promote new version (v%s) of Lambda function %s, it will handle %v percent of traffic", version, fm.Spec.Name, options.Percent)
	return true
}

func configureTrafficRouting(trafficCfg provider.RoutingTrafficConfig, version string, percent int) bool {
	// The primary version has to be set on trafficCfg.
	primary, ok := trafficCfg["primary"]
	if !ok {
		return false
	}
	// Set built version by rollout stage as new primary.
	trafficCfg["primary"] = provider.VersionTraffic{
		Version: version,
		Percent: float64(percent),
	}
	// Make the current primary version as new secondary version in case it's not the latest built version by rollout stage.
	if primary.Version != version {
		trafficCfg["secondary"] = provider.VersionTraffic{
			Version: primary.Version,
			Percent: float64(100 - percent),
		}
	} else {
		// Update traffic to the secondary and keep it as new secondary.
		if secondary, ok := trafficCfg["secondary"]; ok {
			trafficCfg["secondary"] = provider.VersionTraffic{
				Version: secondary.Version,
				Percent: float64(100 - percent),
			}
		}
	}
	return true
}

func build(ctx context.Context, in *executor.Input, client provider.Client, fm provider.FunctionManifest) (version string, ok bool) {
	found, err := client.IsFunctionExist(ctx, fm.Spec.Name)
	if err != nil {
		in.LogPersister.Errorf("Unable to validate function name %s: %v", fm.Spec.Name, err)
		return
	}
	if found {
		if err := client.UpdateFunction(ctx, fm); err != nil {
			in.LogPersister.Errorf("Failed to update lambda function %s: %v", fm.Spec.Name, err)
			return
		}
	} else {
		if err := client.CreateFunction(ctx, fm); err != nil {
			in.LogPersister.Errorf("Failed to create lambda function %s: %v", fm.Spec.Name, err)
			return
		}
	}

	in.LogPersister.Info("Waiting to update lambda function in progress...")
	retry := backoff.NewRetry(provider.RequestRetryTime, backoff.NewConstant(provider.RetryIntervalDuration))
	publishFunctionSucceed := false
	startWaitingStamp := time.Now()
	for retry.WaitNext(ctx) {
		// Commit version for applied Lambda function.
		// Note: via the current docs of [Lambda.PublishVersion](https://docs.aws.amazon.com/sdk-for-go/api/service/lambda/#Lambda.PublishVersion)
		// AWS Lambda doesn't publish a version if the function's configuration and code haven't changed since the last version.
		// But currently, unchanged revision is able to make publish (versionId++) as usual.
		version, err = client.PublishFunction(ctx, fm)
		if err != nil {
			in.Logger.Error("Failed publish new version for Lambda function")
		} else {
			publishFunctionSucceed = true
			break
		}
	}
	if !publishFunctionSucceed {
		in.LogPersister.Errorf("Failed to commit new version for Lambda function %s: %v", fm.Spec.Name, err)
		return
	}

	in.LogPersister.Infof("Successfully committed new version (v%s) for Lambda function %s after duration %v", version, fm.Spec.Name, time.Since(startWaitingStamp))
	ok = true
	return
}
