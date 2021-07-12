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

package piped

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/credentials"

	"github.com/pipe-cd/pipe/pkg/admin"
	"github.com/pipe-cd/pipe/pkg/app/api/service/pipedservice"
	"github.com/pipe-cd/pipe/pkg/app/api/service/pipedservice/pipedclientfake"
	"github.com/pipe-cd/pipe/pkg/app/piped/apistore/applicationstore"
	"github.com/pipe-cd/pipe/pkg/app/piped/apistore/commandstore"
	"github.com/pipe-cd/pipe/pkg/app/piped/apistore/deploymentstore"
	"github.com/pipe-cd/pipe/pkg/app/piped/apistore/environmentstore"
	"github.com/pipe-cd/pipe/pkg/app/piped/apistore/eventstore"
	"github.com/pipe-cd/pipe/pkg/app/piped/chartrepo"
	k8scloudprovidermetrics "github.com/pipe-cd/pipe/pkg/app/piped/cloudprovider/kubernetes/kubernetesmetrics"
	"github.com/pipe-cd/pipe/pkg/app/piped/controller"
	"github.com/pipe-cd/pipe/pkg/app/piped/driftdetector"
	"github.com/pipe-cd/pipe/pkg/app/piped/eventwatcher"
	"github.com/pipe-cd/pipe/pkg/app/piped/livestatereporter"
	"github.com/pipe-cd/pipe/pkg/app/piped/livestatestore"
	k8slivestatestoremetrics "github.com/pipe-cd/pipe/pkg/app/piped/livestatestore/kubernetes/kubernetesmetrics"
	"github.com/pipe-cd/pipe/pkg/app/piped/notifier"
	"github.com/pipe-cd/pipe/pkg/app/piped/planpreview"
	"github.com/pipe-cd/pipe/pkg/app/piped/planpreview/planpreviewmetrics"
	"github.com/pipe-cd/pipe/pkg/app/piped/statsreporter"
	"github.com/pipe-cd/pipe/pkg/app/piped/toolregistry"
	"github.com/pipe-cd/pipe/pkg/app/piped/trigger"
	"github.com/pipe-cd/pipe/pkg/cache/memorycache"
	"github.com/pipe-cd/pipe/pkg/cli"
	"github.com/pipe-cd/pipe/pkg/config"
	"github.com/pipe-cd/pipe/pkg/crypto"
	"github.com/pipe-cd/pipe/pkg/git"
	"github.com/pipe-cd/pipe/pkg/model"
	"github.com/pipe-cd/pipe/pkg/rpc/rpcauth"
	"github.com/pipe-cd/pipe/pkg/rpc/rpcclient"
	"github.com/pipe-cd/pipe/pkg/version"

	// Import to preload all built-in executors to the default registry.
	_ "github.com/pipe-cd/pipe/pkg/app/piped/executor/registry"
	// Import to preload all planners to the default registry.
	_ "github.com/pipe-cd/pipe/pkg/app/piped/planner/registry"
)

type piped struct {
	configFile                           string
	insecure                             bool
	certFile                             string
	adminPort                            int
	toolsDir                             string
	enableDefaultKubernetesCloudProvider bool
	useFakeAPIClient                     bool
	gracePeriod                          time.Duration
	addLoginUserToPasswd                 bool
}

func NewCommand() *cobra.Command {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to detect the current user's home directory: %v", err))
	}
	p := &piped{
		adminPort:   9085,
		toolsDir:    path.Join(home, ".piped", "tools"),
		gracePeriod: 30 * time.Second,
	}
	cmd := &cobra.Command{
		Use:   "piped",
		Short: "Start running piped.",
		RunE:  cli.WithContext(p.run),
	}

	cmd.Flags().StringVar(&p.configFile, "config-file", p.configFile, "The path to the configuration file.")

	cmd.Flags().BoolVar(&p.insecure, "insecure", p.insecure, "Whether disabling transport security while connecting to control-plane.")
	cmd.Flags().StringVar(&p.certFile, "cert-file", p.certFile, "The path to the TLS certificate file.")
	cmd.Flags().IntVar(&p.adminPort, "admin-port", p.adminPort, "The port number used to run a HTTP server for admin tasks such as metrics, healthz.")

	cmd.Flags().StringVar(&p.toolsDir, "tools-dir", p.toolsDir, "The path to directory where to install needed tools such as kubectl, helm, kustomize.")
	cmd.Flags().BoolVar(&p.useFakeAPIClient, "use-fake-api-client", p.useFakeAPIClient, "Whether the fake api client should be used instead of the real one or not.")
	cmd.Flags().BoolVar(&p.enableDefaultKubernetesCloudProvider, "enable-default-kubernetes-cloud-provider", p.enableDefaultKubernetesCloudProvider, "Whether the default kubernetes provider is enabled or not.")
	cmd.Flags().BoolVar(&p.addLoginUserToPasswd, "add-login-user-to-passwd", p.addLoginUserToPasswd, "Whether to add login user to $HOME/passwd. This is typically for applications running as a random user ID.")
	cmd.Flags().DurationVar(&p.gracePeriod, "grace-period", p.gracePeriod, "How long to wait for graceful shutdown.")

	cmd.MarkFlagRequired("config-file")

	return cmd
}

func (p *piped) run(ctx context.Context, t cli.Telemetry) (runErr error) {
	group, ctx := errgroup.WithContext(ctx)
	if p.addLoginUserToPasswd {
		if err := p.insertLoginUserToPasswd(ctx); err != nil {
			return fmt.Errorf("failed to insert logged-in user to passwd: %w", err)
		}
	}

	// Load piped configuration from specified file.
	cfg, err := p.loadConfig()
	if err != nil {
		t.Logger.Error("failed to load piped configuration", zap.Error(err))
		return err
	}

	// Register all metrics.
	registry := registerMetrics(cfg.PipedID)

	// Initialize notifier and add piped events.
	notifier, err := notifier.NewNotifier(cfg, t.Logger)
	if err != nil {
		t.Logger.Error("failed to initialize notifier", zap.Error(err))
		return err
	}
	group.Go(func() error {
		return notifier.Run(ctx)
	})

	// Configure SSH config if needed.
	if cfg.Git.ShouldConfigureSSHConfig() {
		if err := git.AddSSHConfig(cfg.Git); err != nil {
			t.Logger.Error("failed to configure ssh-config", zap.Error(err))
			return err
		}
		t.Logger.Info("successfully configured ssh-config")
	}

	// Initialize default tool registry.
	if err := toolregistry.InitDefaultRegistry(p.toolsDir, t.Logger); err != nil {
		t.Logger.Error("failed to initialize default tool registry", zap.Error(err))
		return err
	}

	// Add configured Helm chart repositories.
	if len(cfg.ChartRepositories) > 0 {
		reg := toolregistry.DefaultRegistry()
		if err := chartrepo.Add(ctx, cfg.ChartRepositories, reg, t.Logger); err != nil {
			t.Logger.Error("failed to add configured chart repositories", zap.Error(err))
			return err
		}
		if len(cfg.ChartRepositories) > 0 {
			if err := chartrepo.Update(ctx, reg, t.Logger); err != nil {
				t.Logger.Error("failed to update Helm chart repositories", zap.Error(err))
				return err
			}
		}
	}

	pipedKey, err := cfg.LoadPipedKey()
	if err != nil {
		t.Logger.Error("failed to load piped key", zap.Error(err))
		return err
	}

	// Make gRPC client and connect to the API.
	apiClient, err := p.createAPIClient(ctx, cfg.APIAddress, cfg.ProjectID, cfg.PipedID, pipedKey, t.Logger)
	if err != nil {
		t.Logger.Error("failed to create gRPC client to control plane", zap.Error(err))
		return err
	}

	// Send the newest piped meta to the control-plane.
	if err := p.sendPipedMeta(ctx, apiClient, cfg, t.Logger); err != nil {
		t.Logger.Error("failed to report piped meta to control-plane", zap.Error(err))
		return err
	}

	// Start running admin server.
	{
		var (
			ver   = []byte(version.Get().Version)
			admin = admin.NewAdmin(p.adminPort, p.gracePeriod, t.Logger)
		)

		admin.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
			w.Write(ver)
		})
		admin.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
		admin.Handle("/metrics", t.PrometheusMetricsHandlerFor(registry))

		group.Go(func() error {
			return admin.Run(ctx)
		})
	}

	// Start running stats reporter.
	{
		url := fmt.Sprintf("http://localhost:%d/metrics", p.adminPort)
		r := statsreporter.NewReporter(url, apiClient, t.Logger)
		group.Go(func() error {
			return r.Run(ctx)
		})
	}

	// Initialize git client.
	gitClient, err := git.NewClient(cfg.Git.Username, cfg.Git.Email, t.Logger)
	if err != nil {
		t.Logger.Error("failed to initialize git client", zap.Error(err))
		return err
	}
	defer func() {
		if err := gitClient.Clean(); err != nil {
			t.Logger.Error("had an error while cleaning gitClient", zap.Error(err))
			return
		}
		t.Logger.Info("successfully cleaned gitClient")
	}()

	// Initialize environment store.
	environmentStore := environmentstore.NewStore(
		apiClient,
		memorycache.NewTTLCache(ctx, 10*time.Minute, time.Minute),
		t.Logger,
	)

	// Start running application store.
	var applicationLister applicationstore.Lister
	{
		store := applicationstore.NewStore(apiClient, p.gracePeriod, t.Logger)
		group.Go(func() error {
			return store.Run(ctx)
		})
		applicationLister = store.Lister()
	}

	// Start running deployment store.
	var deploymentLister deploymentstore.Lister
	{
		store := deploymentstore.NewStore(apiClient, p.gracePeriod, t.Logger)
		group.Go(func() error {
			return store.Run(ctx)
		})
		deploymentLister = store.Lister()
	}

	// Start running command store.
	var commandLister commandstore.Lister
	{
		store := commandstore.NewStore(apiClient, p.gracePeriod, t.Logger)
		group.Go(func() error {
			return store.Run(ctx)
		})
		commandLister = store.Lister()
	}

	// Start running event store.
	var eventGetter eventstore.Getter
	{
		store := eventstore.NewStore(apiClient, p.gracePeriod, t.Logger)
		group.Go(func() error {
			return store.Run(ctx)
		})
		eventGetter = store.Getter()
	}

	// Create memory caches.
	appManifestsCache := memorycache.NewTTLCache(ctx, time.Hour, time.Minute)

	var liveStateGetter livestatestore.Getter
	// Start running application live state store.
	{
		s := livestatestore.NewStore(cfg, applicationLister, p.gracePeriod, t.Logger)
		group.Go(func() error {
			return s.Run(ctx)
		})
		liveStateGetter = s.Getter()
	}

	// Start running application live state reporter.
	{
		r := livestatereporter.NewReporter(applicationLister, liveStateGetter, apiClient, cfg, t.Logger)
		group.Go(func() error {
			return r.Run(ctx)
		})
	}

	decrypter, err := p.initializeSecretDecrypter(cfg)
	if err != nil {
		t.Logger.Error("failed to initialize secret decrypter", zap.Error(err))
		return err
	}

	// Start running application application drift detector.
	{
		d := driftdetector.NewDetector(
			applicationLister,
			gitClient,
			liveStateGetter,
			apiClient,
			appManifestsCache,
			cfg,
			decrypter,
			t.Logger,
		)
		group.Go(func() error {
			return d.Run(ctx)
		})
	}

	// Start running deployment controller.
	{
		c := controller.NewController(
			apiClient,
			gitClient,
			deploymentLister,
			commandLister,
			applicationLister,
			environmentStore,
			livestatestore.LiveResourceLister{Getter: liveStateGetter},
			notifier,
			decrypter,
			cfg,
			appManifestsCache,
			p.gracePeriod,
			t.Logger,
		)

		group.Go(func() error {
			return c.Run(ctx)
		})
	}

	// Start running deployment trigger.
	var lastTriggeredCommitGetter trigger.LastTriggeredCommitGetter
	{
		tr, err := trigger.NewTrigger(
			apiClient,
			gitClient,
			applicationLister,
			commandLister,
			environmentStore,
			notifier,
			cfg,
			p.gracePeriod,
			t.Logger,
		)
		if err != nil {
			t.Logger.Error("failed to initialize trigger", zap.Error(err))
			return err
		}
		lastTriggeredCommitGetter = tr.GetLastTriggeredCommitGetter()

		group.Go(func() error {
			return tr.Run(ctx)
		})
	}

	// Start running event watcher.
	{
		w := eventwatcher.NewWatcher(
			cfg,
			eventGetter,
			gitClient,
			t.Logger,
		)
		group.Go(func() error {
			return w.Run(ctx)
		})
	}

	// Start running planpreview handler.
	{
		// Initialize a dedicated git client for plan-preview feature.
		// Basically, this feature is an utility so it should not share any resource with the main components of piped.
		gc, err := git.NewClient(cfg.Git.Username, cfg.Git.Email, t.Logger)
		if err != nil {
			t.Logger.Error("failed to initialize git client for plan-preview", zap.Error(err))
			return err
		}
		defer func() {
			if err := gc.Clean(); err != nil {
				t.Logger.Error("had an error while cleaning gitClient for plan-preview", zap.Error(err))
				return
			}
			t.Logger.Info("successfully cleaned gitClient for plan-preview")
		}()

		h := planpreview.NewHandler(
			gc,
			apiClient,
			commandLister,
			applicationLister,
			environmentStore,
			lastTriggeredCommitGetter,
			decrypter,
			appManifestsCache,
			cfg,
			planpreview.WithLogger(t.Logger),
		)
		group.Go(func() error {
			return h.Run(ctx)
		})
	}

	// Wait until all piped components have finished.
	// A terminating signal or a finish of any components
	// could trigger the finish of piped.
	// This ensures that all components are good or no one.
	if err := group.Wait(); err != nil {
		t.Logger.Error("failed while running", zap.Error(err))
		return err
	}
	return nil
}

// createAPIClient makes a gRPC client to connect to the API.
func (p *piped) createAPIClient(ctx context.Context, address, projectID, pipedID string, pipedKey []byte, logger *zap.Logger) (pipedservice.Client, error) {
	if p.useFakeAPIClient {
		return pipedclientfake.NewClient(logger), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var (
		token   = rpcauth.MakePipedToken(projectID, pipedID, string(pipedKey))
		creds   = rpcclient.NewPerRPCCredentials(token, rpcauth.PipedTokenCredentials, !p.insecure)
		options = []rpcclient.DialOption{
			rpcclient.WithBlock(),
			rpcclient.WithPerRPCCredentials(creds),
		}
	)

	if !p.insecure {
		if p.certFile != "" {
			options = append(options, rpcclient.WithTLS(p.certFile))
		} else {
			config := &tls.Config{}
			options = append(options, rpcclient.WithTransportCredentials(credentials.NewTLS(config)))
		}
	} else {
		options = append(options, rpcclient.WithInsecure())
	}

	client, err := pipedservice.NewClient(ctx, address, options...)
	if err != nil {
		logger.Error("failed to create api client", zap.Error(err))
		return nil, err
	}
	return client, nil
}

// loadConfig reads the Piped configuration data from the specified file.
func (p *piped) loadConfig() (*config.PipedSpec, error) {
	cfg, err := config.LoadFromYAML(p.configFile)
	if err != nil {
		return nil, err
	}
	if cfg.Kind != config.KindPiped {
		return nil, fmt.Errorf("wrong configuration kind for piped: %v", cfg.Kind)
	}
	if p.enableDefaultKubernetesCloudProvider {
		cfg.PipedSpec.EnableDefaultKubernetesCloudProvider()
	}
	return cfg.PipedSpec, nil
}

func (p *piped) initializeSecretDecrypter(cfg *config.PipedSpec) (crypto.Decrypter, error) {
	sm := cfg.GetSecretManagement()
	if sm == nil {
		return nil, nil
	}

	switch sm.Type {
	case model.SecretManagementTypeNone:
		return nil, nil

	case model.SecretManagementTypeSealingKey:
		fallthrough
	case model.SecretManagementTypeKeyPair:
		if sm.KeyPair.PrivateKeyFile == "" {
			return nil, fmt.Errorf("secretManagement.privateKeyFile must be set")
		}
		decrypter, err := crypto.NewHybridDecrypter(sm.KeyPair.PrivateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize decrypter (%w)", err)
		}
		return decrypter, nil

	case model.SecretManagementTypeGCPKMS:
		return nil, fmt.Errorf("type %q is not implemented yet", sm.Type.String())

	case model.SecretManagementTypeAWSKMS:
		return nil, fmt.Errorf("type %q is not implemented yet", sm.Type.String())

	default:
		return nil, fmt.Errorf("unsupported secret management type: %s", sm.Type.String())
	}
}

func (p *piped) sendPipedMeta(ctx context.Context, client pipedservice.Client, cfg *config.PipedSpec, logger *zap.Logger) error {
	repos := make([]*model.ApplicationGitRepository, 0, len(cfg.Repositories))
	for _, r := range cfg.Repositories {
		repos = append(repos, &model.ApplicationGitRepository{
			Id:     r.RepoID,
			Remote: r.Remote,
			Branch: r.Branch,
		})
	}

	var (
		req = &pipedservice.ReportPipedMetaRequest{
			Version:        version.Get().Version,
			Repositories:   repos,
			CloudProviders: make([]*model.Piped_CloudProvider, 0, len(cfg.CloudProviders)),
		}
		retry = pipedservice.NewRetry(5)
		err   error
	)

	// Configure the list of specified cloud providers.
	for _, cp := range cfg.CloudProviders {
		req.CloudProviders = append(req.CloudProviders, &model.Piped_CloudProvider{
			Name: cp.Name,
			Type: cp.Type.String(),
		})
	}

	// Configure secret management.
	if sm := cfg.GetSecretManagement(); sm != nil {
		switch sm.Type {
		case model.SecretManagementTypeSealingKey:
			fallthrough
		case model.SecretManagementTypeKeyPair:
			publicKey, err := ioutil.ReadFile(sm.KeyPair.PublicKeyFile)
			if err != nil {
				return fmt.Errorf("failed to read public key for secret management (%w)", err)
			}
			req.SecretEncryption = &model.Piped_SecretEncryption{
				Type:      sm.Type.String(),
				PublicKey: string(publicKey),
			}
		}
	}
	if req.SecretEncryption == nil {
		req.SecretEncryption = &model.Piped_SecretEncryption{
			Type: model.SecretManagementTypeNone.String(),
		}
	}

	for retry.WaitNext(ctx) {
		if _, err = client.ReportPipedMeta(ctx, req); err == nil {
			return nil
		}
		logger.Warn("failed to report piped meta to control-plane, wait to the next retry",
			zap.Int("calls", retry.Calls()),
			zap.Error(err),
		)
	}

	return err
}

// insertLoginUserToPasswd adds the logged-in user to /etc/passwd.
// It requires nss_wrapper (https://cwrap.org/nss_wrapper.html)
// to get the operation done.
//
// This is a workaround to deal with OpenShift less than 4.2
// See more: https://github.com/pipe-cd/pipe/issues/1905
func (p *piped) insertLoginUserToPasswd(ctx context.Context) error {
	var stdout, stderr bytes.Buffer

	// Use the id command so that it gets proper ids even in pure Go.
	cmd := exec.CommandContext(ctx, "id", "-u")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get uid: %s", &stderr)
	}
	uid := strings.TrimSpace(stdout.String())

	stdout.Reset()
	stderr.Reset()

	cmd = exec.CommandContext(ctx, "id", "-g")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get gid: %s", &stderr)
	}
	gid := strings.TrimSpace(stdout.String())

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to detect the current user's home directory: %w", err)
	}

	// echo "default:x:${USER_ID}:${GROUP_ID}:Dynamically created user:${HOME}:/sbin/nologin" >> "$HOME/passwd"
	entry := fmt.Sprintf("\ndefault:x:%s:%s:Dynamically created user:%s:/sbin/nologin", uid, gid, home)
	nssPasswdPath := filepath.Join(home, "passwd")
	f, err := os.OpenFile(nssPasswdPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", nssPasswdPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to append entry to %q: %w", nssPasswdPath, err)
	}

	return nil
}

func registerMetrics(pipedID string) *prometheus.Registry {
	r := prometheus.NewRegistry()
	wrapped := prometheus.WrapRegistererWith(
		prometheus.Labels{
			"piped":         pipedID,
			"piped_version": version.Get().Version,
		},
		r,
	)
	wrapped.Register(prometheus.NewGoCollector())
	wrapped.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	k8scloudprovidermetrics.Register(wrapped)
	k8slivestatestoremetrics.Register(wrapped)
	planpreviewmetrics.Register(wrapped)

	return r
}
