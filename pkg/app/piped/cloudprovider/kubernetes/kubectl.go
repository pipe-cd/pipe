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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/client-go/rest"
)

type Kubectl struct {
	version  string
	execPath string
	config   *rest.Config
}

func NewKubectl(version, path string) *Kubectl {
	return &Kubectl{
		version:  version,
		execPath: path,
	}
}

func (c *Kubectl) Apply(ctx context.Context, manifest Manifest) (err error) {
	defer func() {
		metricsKubectlCalled(c.version, "apply", err == nil)
	}()

	data, err := manifest.YamlBytes()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, c.execPath, "apply", "-f", "-")
	r := bytes.NewReader(data)
	cmd.Stdin = r

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply: %s (%v)", string(out), err)
	}
	return nil
}

func (c *Kubectl) Delete(ctx context.Context, r ResourceKey) (err error) {
	defer func() {
		metricsKubectlCalled(c.version, "delete", err == nil)
	}()

	args := []string{"delete", r.Kind, r.Name}
	if r.Namespace != "" {
		args = append(args, "-n", r.Namespace)
	}
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	out, err := cmd.CombinedOutput()

	if strings.Contains(string(out), "(NotFound)") {
		return fmt.Errorf("failed to delete: %s, (%w), %v", string(out), ErrNotFound, err)
	}
	if err != nil {
		return fmt.Errorf("failed to delete: %s, %v", string(out), err)
	}
	return nil
}
