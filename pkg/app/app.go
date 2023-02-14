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
	"github.com/fairwindsops/vault-token-injector/pkg/tfcloud"
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
)

// App is the main application struct
type App struct {
	Config         *Config
	CircleToken    string
	VaultTokenFile string
	VaultClient    *vault.Client
	TFCloudToken   string
	EnableMetrics  bool
	Errors         *Errors
}

// Config represents the configuration file
type Config struct {
	CircleCI []CircleCIConfig `mapstructure:"circleci"`
	TFCloud  []TFCloudConfig  `mapstructure:"tfcloud"`
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

// NewApp creates a new App from the given configuration options
func NewApp(circleToken, vaultTokenFile, tfCloudToken string, config *Config, enableMetrics bool) *App {
	app := &App{
		Config:         config,
		CircleToken:    circleToken,
		TFCloudToken:   tfCloudToken,
		VaultTokenFile: vaultTokenFile,
		EnableMetrics:  enableMetrics,
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
		a.registerErrors()
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(":4329", nil)
	}

	klog.Info("starting main application loop")
	for {
		if err := a.refreshVaultToken(); err != nil {
			klog.Errorf("unable to get a valid token, skipping loop: %s", err)
			a.incrementVaultError()
			time.Sleep(a.Config.TokenRefreshInterval)
			continue
		}
		var wg sync.WaitGroup
		for _, workspace := range a.Config.TFCloud {

			wg.Add(1)
			go a.updateTFCloudInstance(workspace, &wg)
		}
		for _, project := range a.Config.CircleCI {
			wg.Add(1)
			go a.updateCircleCIInstance(project, &wg)
		}
		wg.Wait()

		time.Sleep(a.Config.TokenRefreshInterval)
	}
}

func (a *App) updateCircleCIInstance(project CircleCIConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	projName := project.Name
	projVariableName := a.Config.TokenVariable
	token, err := a.VaultClient.CreateToken(project.VaultRole, project.VaultPolicies, a.Config.TokenTTL, a.Config.OrphanTokens)
	if err != nil {
		a.incrementVaultError()
		klog.Errorf("error making token for CircleCI project %s: %w", projName, err)
		return
	}
	klog.V(10).Infof("got token %s for CircleCI project %s", token.Auth.ClientToken, projName)
	klog.Infof("setting env var %s to vault token value in CircleCI project %s", projVariableName, projName)
	if err := circleci.UpdateEnvVar(projName, projVariableName, token.Auth.ClientToken, a.CircleToken); err != nil {
		a.incremenCircleCIError()
		klog.Errorf("error updating CircleCI project %s with token value: %w", projName, err)
		return
	}
	if err := circleci.UpdateEnvVar(projName, "VAULT_ADDR", a.Config.VaultAddress, a.CircleToken); err != nil {
		a.incremenCircleCIError()
		klog.Errorf("error updating VAULT_ADDR in CircleCI project %s: %w", projName, err)
		return
	}
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
		klog.Errorf("error getting vault token for TFCloud workspace %s: %w", workspaceLogIdentifier, err)
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
		klog.Errorf("error updating token for TFCloud workspace %s: %w", workspaceLogIdentifier, err)
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
		klog.Errorf("error updating VAULT_ADDR for ws %s: %w", workspaceLogIdentifier, err)
		return
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
