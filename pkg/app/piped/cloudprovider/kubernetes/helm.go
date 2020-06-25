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
	"path/filepath"

	"github.com/pipe-cd/pipe/pkg/config"
)

type Helm struct {
	version  string
	execPath string
}

func NewHelm(version, path string) *Helm {
	return &Helm{
		version:  version,
		execPath: path,
	}
}

func (c *Helm) Template(ctx context.Context, appName, appDir string, chart *config.InputHelmChart, opts *config.InputHelmOptions) (string, error) {
	releaseName := appName
	if opts != nil && opts.ReleaseName != "" {
		releaseName = opts.ReleaseName
	}

	// TODO: Support remote git chart and remote helm chart.
	args := []string{
		"template",
		"--no-hooks",
		releaseName,
		filepath.Join(appDir, chart.Path),
	}
	cmd := exec.CommandContext(ctx, c.execPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
