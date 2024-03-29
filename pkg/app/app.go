package app

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/vault-token-injector/pkg/circleci"
	"github.com/fairwindsops/vault-token-injector/pkg/spacelift"
	"github.com/fairwindsops/vault-token-injector/pkg/tfcloud"
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
)

// App is the main application struct
type App struct {
	Config          *Config
	CircleToken     string
	VaultTokenFile  string
	VaultClient     *vault.Client
	TFCloudToken    string
	EnableMetrics   bool
	Metrics         *Metrics
	SpaceliftClient *spacelift.Client
}

// Config represents the configuration file
type Config struct {
	CircleCI  []CircleCIConfig  `mapstructure:"circleci"`
	TFCloud   []TFCloudConfig   `mapstructure:"tfcloud"`
	Spacelift []SpaceliftConfig `mapstructure:"spacelift"`
	// The address of the vault server to use when creating tokens
	VaultAddress string `mapstructure:"vault_address"`
	// The variable name to use when setting a vault token. Defaults to VAULT_ADDR
	TokenVariable string `mapstructure:"token_variable"`
	// If true, all tokens will be created with the orphan flag set to true
	OrphanTokens bool `mapstructure:"orphan_tokens"`
	// The TTL of the tokens that will be created. Defaults to 30 minutes
	TokenTTL time.Duration `mapstructure:"token_ttl"`
	// The interval at which the token will be refreshed. Defaults to 1 hour
	TokenRefreshInterval time.Duration `mapstructure:"token_refresh_interval"`
}

// CircleCIConfig represents a specific instance of a CircleCI project we want to
// update an environment variable for
type CircleCIConfig struct {
	Name          string   `mapstructure:"name"`
	VaultRole     *string  `mapstructure:"vault_role"`
	VaultPolicies []string `mapstructure:"vault_policies"`
}

// TFCloudConfig represents a specific instance of a TFCloud workspace we want to
// update an environment variable for
type TFCloudConfig struct {
	// Workspace is the ID of the workspace in tfcloud. Should begin with ws- and is required
	Workspace string `mapstructure:"workspace"`
	// Name is an optional field that can be used to identify a workspace
	Name string `mapstructure:"name"`
	// VaultRole is the vault role to use for the token in this workspace
	VaultRole *string `mapstructure:"vault_role"`
	// VaultPolicies is a list of policies that will be given to the token in this workspace
	VaultPolicies []string `mapstructure:"vault_policies"`
}

type SpaceliftConfig struct {
	// Stack is the name of a Spacelift stack that you want to inject vars into
	Stack string `mapstructure:"stack"`
	// VaultRole is the vault role to use for the token in this stack
	VaultRole *string `mapstructure:"vault_role"`
	// VaultPolicies is a list of policies that will be given to the token in this stack
	VaultPolicies []string `mapstructure:"vault_policies"`
}

// NewApp creates a new App from the given configuration options
func NewApp(circleToken, vaultTokenFile, tfCloudToken string, config *Config, enableMetrics bool, spaceliftClient *spacelift.Client) *App {
	app := &App{
		Config:          config,
		CircleToken:     circleToken,
		TFCloudToken:    tfCloudToken,
		VaultTokenFile:  vaultTokenFile,
		SpaceliftClient: spaceliftClient,
		EnableMetrics:   enableMetrics,
	}

	if len(app.Config.CircleCI) > 0 && circleToken == "" {
		klog.Error("CircleCI is configured but no token was provided.")
	}

	if len(app.Config.TFCloud) > 0 && tfCloudToken == "" {
		klog.Error("TFCloud is configured but no token was provided.")
	}
	if app.Config.TokenVariable == "" {
		app.Config.TokenVariable = "VAULT_TOKEN"
		klog.Warningf("token variable not set, defaulting to %s", app.Config.TokenVariable)
	}

	if app.Config.TokenTTL == 0 {
		app.Config.TokenTTL = time.Minute * 60
		klog.V(4).Infof("token TTL not set, defaulting to %s", app.Config.TokenTTL.String())
	}

	if app.Config.TokenRefreshInterval == 0 {
		app.Config.TokenRefreshInterval = time.Minute * 30
		klog.V(3).Infof("token refresh interval not set, defaulting to %s", app.Config.TokenRefreshInterval.String())
	}

	klog.V(3).Infof("Token Variable: %s", app.Config.TokenVariable)
	klog.V(3).Infof("Token TTL: %s", app.Config.TokenTTL.String())
	klog.V(3).Infof("Token Refresh Interval: %s", app.Config.TokenRefreshInterval.String())
	klog.V(3).Infof("Vault Address: %s", app.Config.VaultAddress)
	klog.V(3).Infof("Orphan Tokens: %t", app.Config.OrphanTokens)
	klog.V(3).Infof("Circle Configs: %v", app.Config.CircleCI)
	klog.V(3).Infof("TFCloud Configs: %v", app.Config.TFCloud)

	return app
}

// Run starts the application
func (a *App) Run() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		klog.Info("exiting - received termination signal")
		os.Exit(0)
	}()

	if a.EnableMetrics {
		a.registerMetrics()
		http.Handle("/metrics", promhttp.Handler())
		http.Handle("/health", http.HandlerFunc(a.healthHandler))
		go http.ListenAndServe(":4329", nil)
	}

	klog.Info("starting main application loop")
	for {
		var wg sync.WaitGroup
		if err := a.injectVars(&wg); err != nil {
			time.Sleep(a.Config.TokenRefreshInterval)
			continue
		}
		wg.Wait()

		time.Sleep(a.Config.TokenRefreshInterval)
	}
}

// RunOnce just does a single run for use with a Kubernetes cronjob
func (a *App) RunOnce() error {

	klog.Info("running the token injection once")

	a.registerMetrics()
	if a.EnableMetrics {
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(":4329", nil)
	}

	var wg sync.WaitGroup
	if err := a.injectVars(&wg); err != nil {
		return err
	}
	wg.Wait()
	errCount := getMetricValue(a.Metrics.totalErrorCount)
	if errCount > 0 {
		return fmt.Errorf("there were errors during during the run. see the logs for more details")
	}
	return nil
}

func (a *App) injectVars(wg *sync.WaitGroup) error {
	if err := a.refreshVaultToken(); err != nil {
		klog.Errorf("unable to get a valid token, skipping loop: %s", err)
		a.incrementVaultError()
		return err
	}
	for _, workspace := range a.Config.TFCloud {
		wg.Add(1)
		go a.updateTFCloudInstance(workspace, wg)
	}
	for _, project := range a.Config.CircleCI {
		wg.Add(1)
		go a.updateCircleCIInstance(project, wg)
	}
	for _, stack := range a.Config.Spacelift {
		wg.Add(1)
		go a.updateSpaceliftInstance(stack, wg)
	}
	return nil
}

func (a *App) updateCircleCIInstance(project CircleCIConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	projName := project.Name
	projVariableName := a.Config.TokenVariable
	token, err := a.VaultClient.CreateToken(project.VaultRole, project.VaultPolicies, a.Config.TokenTTL, a.Config.OrphanTokens)
	if err != nil {
		a.incrementVaultError()
		klog.Errorf("error making token for CircleCI project %s: %s", projName, err.Error())
		return
	}
	klog.V(10).Infof("got token %s for CircleCI project %s", token.Auth.ClientToken, projName)
	klog.Infof("setting env var %s to vault token value in CircleCI project %s", projVariableName, projName)
	if err := circleci.UpdateEnvVar(projName, projVariableName, token.Auth.ClientToken, a.CircleToken); err != nil {
		a.incrementCircleCIError()
		klog.Errorf("error updating CircleCI project %s with token value: %s", projName, err.Error())
		return
	}
	if err := circleci.UpdateEnvVar(projName, "VAULT_ADDR", a.Config.VaultAddress, a.CircleToken); err != nil {
		a.incrementCircleCIError()
		klog.Errorf("error updating VAULT_ADDR in CircleCI project %s: %s", projName, err.Error())
		return
	}
	if a.EnableMetrics {
		a.Metrics.circleTokensUpdated.Inc()
	}
}

func (a *App) updateSpaceliftInstance(instance SpaceliftConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := a.SpaceliftClient.RefreshJWT(); err != nil {
		klog.Errorf("could not refresh Spacelift API auth via JWT: %s", err.Error())
		a.incrementSpaceliftError()
		return
	}

	token, err := a.VaultClient.CreateToken(instance.VaultRole, instance.VaultPolicies, a.Config.TokenTTL, a.Config.OrphanTokens)
	if err != nil {
		a.incrementVaultError()
		klog.Errorf("error getting vault token for spacelift stack %s: %s", instance.Stack, err.Error())
		return
	}
	klog.V(10).Infof("got token %s for spacelift stack %s", token.Auth.ClientToken, instance.Stack)

	envVars := []spacelift.EnvVar{
		{
			Key:       "VAULT_ADDR",
			Value:     a.Config.VaultAddress,
			WriteOnly: false,
		},
		{
			Key:       a.Config.TokenVariable,
			Value:     token.Auth.ClientToken,
			WriteOnly: true,
		},
	}
	if err := a.SpaceliftClient.SetEnvVars(instance.Stack, envVars); err != nil {
		a.incrementSpaceliftError()
		klog.Errorf("error setting variables in Spacelift stack %s: %s", instance.Stack, err.Error())
		return
	}
	klog.Infof("successfully updated spacelift vars in stack: %s", instance.Stack)
}

func (a *App) updateTFCloudInstance(instance TFCloudConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	var workspaceLogIdentifier string
	if instance.Name != "" {
		workspaceLogIdentifier = instance.Name
	} else {
		workspaceLogIdentifier = instance.Workspace
	}
	token, err := a.VaultClient.CreateToken(instance.VaultRole, instance.VaultPolicies, a.Config.TokenTTL, a.Config.OrphanTokens)
	if err != nil {
		a.incrementVaultError()
		klog.Errorf("error getting vault token for TFCloud workspace %s: %s", workspaceLogIdentifier, err.Error())
		return
	}
	klog.V(10).Infof("got token %v for tfcloud workspace %s", token.Auth.ClientToken, workspaceLogIdentifier)
	klog.Infof("setting env var %s to vault token value", a.Config.TokenVariable)
	tokenVar := tfcloud.Variable{
		Key:       a.Config.TokenVariable,
		Value:     token.Auth.ClientToken,
		Token:     a.TFCloudToken,
		Sensitive: true,
		Workspace: instance.Workspace,
	}
	if err := tokenVar.Update(); err != nil {
		a.incrementTfCloudError()
		klog.Errorf("error updating token for TFCloud workspace %s: %s", workspaceLogIdentifier, err.Error())
		return
	}
	addressVar := tfcloud.Variable{
		Key:                 "VAULT_ADDR",
		Value:               a.Config.VaultAddress,
		Sensitive:           false,
		Token:               a.TFCloudToken,
		Workspace:           instance.Workspace,
		WorkspaceIdentifier: workspaceLogIdentifier,
	}
	if err := addressVar.Update(); err != nil {
		a.incrementTfCloudError()
		klog.Errorf("error updating VAULT_ADDR for ws %s: %s", workspaceLogIdentifier, err.Error())
		return
	}
	if a.EnableMetrics {
		a.Metrics.tfcloudTokensUpdated.Inc()
	}
}

func (a *App) refreshVaultToken() error {
	var client *vault.Client
	if a.VaultTokenFile != "" {
		klog.V(3).Infof("attempting to refresh token from file")
		tokenData, err := os.ReadFile(a.VaultTokenFile)
		if err != nil {
			return fmt.Errorf("vault-token-file is set but could not be opened: %s", err.Error())
		}
		token := strings.TrimSpace(string(tokenData))
		if err := os.Setenv("VAULT_TOKEN", token); err != nil {
			return fmt.Errorf("could not set VAULT_TOKEN from file: %s", err.Error())
		}
		client, err = vault.NewClient(a.Config.VaultAddress, token)
		if err != nil {
			return err
		}
	} else {
		var err error
		client, err = vault.NewClient(a.Config.VaultAddress, os.Getenv("VAULT_TOKEN"))
		if err != nil {
			return err
		}
	}
	if err := client.LookupSelf(); err != nil {
		klog.V(4).Infof("error looking up self: %s", err.Error())
		return fmt.Errorf("current token was unable to lookup self, assuming invalid")
	}
	a.VaultClient = client
	return nil
}
