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

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	provider "github.com/pipe-cd/pipe/pkg/app/piped/cloudprovider/kubernetes"
	"github.com/pipe-cd/pipe/pkg/app/piped/executor"
	"github.com/pipe-cd/pipe/pkg/cache"
	"github.com/pipe-cd/pipe/pkg/config"
	"github.com/pipe-cd/pipe/pkg/model"
)

const (
	variantLabel = "pipecd.dev/variant" // Variant name: primary, stage, baseline
)

type deployExecutor struct {
	executor.Input

	commit    string
	deployCfg *config.KubernetesDeploymentSpec
	provider  provider.Provider
}

type registerer interface {
	Register(stage model.Stage, f executor.Factory) error
	RegisterRollback(kind model.ApplicationKind, f executor.Factory) error
}

// Register registers this executor factory into a given registerer.
func Register(r registerer) {
	f := func(in executor.Input) executor.Executor {
		return &deployExecutor{
			Input: in,
		}
	}

	r.Register(model.StageK8sSync, f)
	r.Register(model.StageK8sPrimaryRollout, f)
	r.Register(model.StageK8sCanaryRollout, f)
	r.Register(model.StageK8sCanaryClean, f)
	r.Register(model.StageK8sBaselineRollout, f)
	r.Register(model.StageK8sBaselineClean, f)
	r.Register(model.StageK8sTrafficRouting, f)

	r.RegisterRollback(model.ApplicationKind_KUBERNETES, func(in executor.Input) executor.Executor {
		return &rollbackExecutor{
			Input: in,
		}
	})
}

func (e *deployExecutor) Execute(sig executor.StopSignal) model.StageStatus {
	ctx := sig.Context()
	e.commit = e.Deployment.Trigger.Commit.Hash

	ds, err := e.TargetDSP.Get(ctx, e.LogPersister)
	if err != nil {
		e.LogPersister.Errorf("Failed to prepare target deploy source data (%v)", err)
		return model.StageStatus_STAGE_FAILURE
	}

	e.deployCfg = ds.DeploymentConfig.KubernetesDeploymentSpec
	if e.deployCfg == nil {
		e.LogPersister.Error("Malformed deployment configuration: missing KubernetesDeploymentSpec")
		return model.StageStatus_STAGE_FAILURE
	}

	e.provider = provider.NewProvider(e.Deployment.ApplicationName, ds.AppDir, ds.RepoDir, e.Deployment.GitPath.ConfigFilename, e.deployCfg.Input, e.Logger)
	e.Logger.Info("start executing kubernetes stage",
		zap.String("stage-name", e.Stage.Name),
		zap.String("app-dir", ds.AppDir),
	)

	var (
		originalStatus = e.Stage.Status
		status         model.StageStatus
	)

	switch model.Stage(e.Stage.Name) {
	case model.StageK8sSync:
		status = e.ensureSync(ctx)

	case model.StageK8sPrimaryRollout:
		status = e.ensurePrimaryRollout(ctx)

	case model.StageK8sCanaryRollout:
		status = e.ensureCanaryRollout(ctx)

	case model.StageK8sCanaryClean:
		status = e.ensureCanaryClean(ctx)

	case model.StageK8sBaselineRollout:
		status = e.ensureBaselineRollout(ctx)

	case model.StageK8sBaselineClean:
		status = e.ensureBaselineClean(ctx)

	case model.StageK8sTrafficRouting:
		status = e.ensureTrafficRouting(ctx)

	default:
		e.LogPersister.Errorf("Unsupported stage %s for kubernetes application", e.Stage.Name)
		return model.StageStatus_STAGE_FAILURE
	}

	return executor.DetermineStageStatus(sig.Signal(), originalStatus, status)
}

func (e *deployExecutor) loadRunningManifests(ctx context.Context) (manifests []provider.Manifest, err error) {
	commit := e.Deployment.RunningCommitHash
	if commit == "" {
		return nil, fmt.Errorf("unable to determine running commit")
	}

	loader := &manifestsLoadFunc{
		loadFunc: func(ctx context.Context) ([]provider.Manifest, error) {
			ds, err := e.RunningDSP.Get(ctx, e.LogPersister)
			if err != nil {
				e.LogPersister.Errorf("Failed to prepare running deploy source (%v)", err)
			}

			loader := provider.NewManifestLoader(
				e.Deployment.ApplicationName,
				ds.AppDir,
				ds.RepoDir,
				e.Deployment.GitPath.ConfigFilename,
				e.deployCfg.Input,
				e.Logger,
			)
			return loader.LoadManifests(ctx)
		},
	}

	return loadManifests(ctx, e.Deployment.ApplicationId, commit, e.AppManifestsCache, loader, e.Logger)
}

type manifestsLoadFunc struct {
	loadFunc func(context.Context) ([]provider.Manifest, error)
}

func (l *manifestsLoadFunc) LoadManifests(ctx context.Context) ([]provider.Manifest, error) {
	return l.loadFunc(ctx)
}

func loadManifests(ctx context.Context, appID, commit string, manifestsCache cache.Cache, loader provider.ManifestLoader, logger *zap.Logger) (manifests []provider.Manifest, err error) {
	cache := provider.AppManifestsCache{
		AppID:  appID,
		Cache:  manifestsCache,
		Logger: logger,
	}
	manifests, ok := cache.Get(commit)
	if ok {
		return manifests, nil
	}

	// When the manifests were not in the cache we have to load them.
	manifests, err = loader.LoadManifests(ctx)
	if err != nil {
		return nil, err
	}
	cache.Put(commit, manifests)

	return manifests, nil
}

func addBuiltinAnnontations(manifests []provider.Manifest, variant, hash, pipedID, appID string) {
	for i := range manifests {
		manifests[i].AddAnnotations(map[string]string{
			provider.LabelManagedBy:          provider.ManagedByPiped,
			provider.LabelPiped:              pipedID,
			provider.LabelApplication:        appID,
			variantLabel:                     variant,
			provider.LabelOriginalAPIVersion: manifests[i].Key.APIVersion,
			provider.LabelResourceKey:        manifests[i].Key.String(),
			provider.LabelCommitHash:         hash,
		})
	}
}

func applyManifests(ctx context.Context, applier provider.Applier, manifests []provider.Manifest, namespace string, lp executor.LogPersister) error {
	if namespace == "" {
		lp.Infof("Start applying %d manifests", len(manifests))
	} else {
		lp.Infof("Start applying %d manifests to %q namespace", len(manifests), namespace)
	}
	for _, m := range manifests {
		if err := applier.ApplyManifest(ctx, m); err != nil {
			lp.Errorf("Failed to apply manifest: %s (%v)", m.Key.ReadableString(), err)
			return err
		}
		lp.Successf("- applied manifest: %s", m.Key.ReadableString())
	}
	lp.Successf("Successfully applied %d manifests", len(manifests))
	return nil
}

func deleteResources(ctx context.Context, applier provider.Applier, resources []provider.ResourceKey, lp executor.LogPersister) error {
	resourcesLen := len(resources)
	if resourcesLen == 0 {
		lp.Info("No resources to delete")
		return nil
	}

	lp.Infof("Start deleting %d resources", len(resources))
	var deletedCount int

	for _, k := range resources {
		err := applier.Delete(ctx, k)
		if err == nil {
			lp.Successf("- deleted resource: %s", k.ReadableString())
			deletedCount++
			continue
		}
		if errors.Is(err, provider.ErrNotFound) {
			lp.Infof("- no resource %s to delete", k.ReadableString())
			deletedCount++
			continue
		}
		lp.Errorf("- unable to delete resource: %s (%v)", k.ReadableString(), err)
	}

	if deletedCount < resourcesLen {
		lp.Infof("Deleted %d/%d resources", deletedCount, resourcesLen)
		return fmt.Errorf("unable to delete %d resources", resourcesLen-deletedCount)
	}

	lp.Successf("Successfully deleted %d resources", len(resources))
	return nil
}

func findManifests(kind, name string, manifests []provider.Manifest) []provider.Manifest {
	var out []provider.Manifest
	for _, m := range manifests {
		if m.Key.Kind != kind {
			continue
		}
		if name != "" && m.Key.Name != name {
			continue
		}
		out = append(out, m)
	}
	return out
}

func findConfigMapManifests(manifests []provider.Manifest) []provider.Manifest {
	var out []provider.Manifest
	for _, m := range manifests {
		if !m.Key.IsConfigMap() {
			continue
		}
		out = append(out, m)
	}
	return out
}

func findSecretManifests(manifests []provider.Manifest) []provider.Manifest {
	var out []provider.Manifest
	for _, m := range manifests {
		if !m.Key.IsSecret() {
			continue
		}
		out = append(out, m)
	}
	return out
}

func findWorkloadManifests(manifests []provider.Manifest, refs []config.K8sResourceReference) []provider.Manifest {
	if len(refs) == 0 {
		return findManifests(provider.KindDeployment, "", manifests)
	}

	workloads := make([]provider.Manifest, 0)
	for _, ref := range refs {
		kind := provider.KindDeployment
		if ref.Kind != "" {
			kind = ref.Kind
		}
		ms := findManifests(kind, ref.Name, manifests)
		workloads = append(workloads, ms...)
	}
	return workloads
}

func duplicateManifests(manifests []provider.Manifest, nameSuffix string) []provider.Manifest {
	out := make([]provider.Manifest, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, duplicateManifest(m, nameSuffix))
	}
	return out
}

func duplicateManifest(m provider.Manifest, nameSuffix string) provider.Manifest {
	name := makeSuffixedName(m.Key.Name, nameSuffix)
	return m.Duplicate(name)
}

func generateVariantServiceManifests(services []provider.Manifest, variant, nameSuffix string) ([]provider.Manifest, error) {
	manifests := make([]provider.Manifest, 0, len(services))
	updateService := func(s *corev1.Service) {
		s.Name = makeSuffixedName(s.Name, nameSuffix)
		// Currently, we suppose that all generated services should be ClusterIP.
		s.Spec.Type = corev1.ServiceTypeClusterIP
		// Append the variant label to the selector
		// to ensure that the generated service is using only workloads of this variant.
		if s.Spec.Selector == nil {
			s.Spec.Selector = map[string]string{}
		}
		s.Spec.Selector[variantLabel] = variant
		// Empty all unneeded fields.
		s.Spec.ExternalIPs = nil
		s.Spec.LoadBalancerIP = ""
		s.Spec.LoadBalancerSourceRanges = nil
	}

	for _, m := range services {
		s := &corev1.Service{}
		if err := m.ConvertToStructuredObject(s); err != nil {
			return nil, err
		}
		updateService(s)
		manifest, err := provider.ParseFromStructuredObject(s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Service object to Manifest: %w", err)
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func generateVariantWorkloadManifests(workloads, configmaps, secrets []provider.Manifest, variant, nameSuffix string, replicasCalculator func(*int32) int32) ([]provider.Manifest, error) {
	manifests := make([]provider.Manifest, 0, len(workloads))

	cmNames := make(map[string]struct{}, len(configmaps))
	for i := range configmaps {
		cmNames[configmaps[i].Key.Name] = struct{}{}
	}

	secretNames := make(map[string]struct{}, len(secrets))
	for i := range secrets {
		secretNames[secrets[i].Key.Name] = struct{}{}
	}

	updatePod := func(pod *corev1.PodTemplateSpec) {
		// Add variant labels.
		if pod.Labels == nil {
			pod.Labels = map[string]string{}
		}
		pod.Labels[variantLabel] = variant

		// Update volumes to use canary's ConfigMaps and Secrets.
		for i := range pod.Spec.Volumes {
			if cm := pod.Spec.Volumes[i].ConfigMap; cm != nil {
				if _, ok := cmNames[cm.Name]; ok {
					cm.Name = makeSuffixedName(cm.Name, nameSuffix)
				}
			}
			if s := pod.Spec.Volumes[i].Secret; s != nil {
				if _, ok := secretNames[s.SecretName]; ok {
					s.SecretName = makeSuffixedName(s.SecretName, nameSuffix)
				}
			}
		}
	}

	updateDeployment := func(d *appsv1.Deployment) {
		d.Name = makeSuffixedName(d.Name, nameSuffix)
		if replicasCalculator != nil {
			replicas := replicasCalculator(d.Spec.Replicas)
			d.Spec.Replicas = &replicas
		}
		d.Spec.Selector = metav1.AddLabelToSelector(d.Spec.Selector, variantLabel, variant)
		updatePod(&d.Spec.Template)
	}

	for _, m := range workloads {
		switch m.Key.Kind {
		case provider.KindDeployment:
			d := &appsv1.Deployment{}
			if err := m.ConvertToStructuredObject(d); err != nil {
				return nil, err
			}
			updateDeployment(d)
			manifest, err := provider.ParseFromStructuredObject(d)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, manifest)

		default:
			return nil, fmt.Errorf("unsupported workload kind %s", m.Key.Kind)
		}
	}

	return manifests, nil
}

func checkVariantSelectorInWorkload(m provider.Manifest, variant string) error {
	var (
		matchLabelsFields = []string{"spec", "selector", "matchLabels"}
		labelsFields      = []string{"spec", "template", "metadata", "labels"}
	)

	matchLabels, err := m.GetNestedStringMap(matchLabelsFields...)
	if err != nil {
		return err
	}
	value, ok := matchLabels[variantLabel]
	if !ok {
		return fmt.Errorf("missing %s key in spec.selector.matchLabels", variantLabel)
	}
	if value != variant {
		return fmt.Errorf("require %s but got %s for %s key in %s", variant, value, variantLabel, strings.Join(matchLabelsFields, "."))
	}

	labels, err := m.GetNestedStringMap(labelsFields...)
	if err != nil {
		return err
	}
	value, ok = labels[variantLabel]
	if !ok {
		return fmt.Errorf("missing %s key in spec.template.metadata.labels", variantLabel)
	}
	if value != variant {
		return fmt.Errorf("require %s but got %s for %s key in %s", variant, value, variantLabel, strings.Join(labelsFields, "."))
	}

	return nil
}

func ensureVariantSelectorInWorkload(m provider.Manifest, variant string) error {
	variantMap := map[string]string{
		variantLabel: variant,
	}
	if err := m.AddStringMapValues(variantMap, "spec", "selector", "matchLabels"); err != nil {
		return err
	}
	return m.AddStringMapValues(variantMap, "spec", "template", "metadata", "labels")
}

func makeSuffixedName(name, suffix string) string {
	if suffix != "" {
		return name + "-" + suffix
	}
	return name
}
