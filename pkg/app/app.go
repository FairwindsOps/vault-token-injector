package app

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	"github.com/fairwindsops/vault-token-injector/pkg/circleci"
	"github.com/fairwindsops/vault-token-injector/pkg/tfcloud"
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
)

type App struct {
	Config         *Config
	CircleToken    string
	VaultTokenFile string
	TFCloudToken   string
}

// Config represents the top level our applications config yaml file
type Config struct {
	CircleCI      []CircleCIConfig `mapstructure:"circleci"`
	TFCloud       []TFCloudConfig  `mapstructure:"tfcloud"`
	VaultAddress  string           `mapstructure:"vault_address"`
	TokenVariable string           `mapstructure:"token_variable"`
}

// CircleCIConfig represents a specific instance of a CircleCI project we want to
// update an environment variable for
type CircleCIConfig struct {
	Name      string `mapstructure:"name"`
	VaultRole string `mapstructure:"vault_role"`
}

// TFCloudConfig represents a specific instance of a TFCloud workspace we want to
// update an environment variable for
type TFCloudConfig struct {
	Workspace string `mapstructure:"workspace"`
	VaultRole string `mapstructure:"vault_role"`
}

func NewApp(circleToken, vaultTokenFile, tfCloudToken string, config *Config) *App {
	app := &App{
		Config:         config,
		CircleToken:    circleToken,
		TFCloudToken:   tfCloudToken,
		VaultTokenFile: vaultTokenFile,
	}
	if len(app.Config.CircleCI) > 0 && circleToken == "" {
		klog.Warning("CircleCI is configured but no token was provided.")
	}

	if len(app.Config.TFCloud) > 0 && tfCloudToken == "" {
		klog.Warning("TFCloud is configured but no token was provided.")
	}
	if app.Config.TokenVariable == "" {
		app.Config.TokenVariable = "VAULT_TOKEN"
		klog.Warningf("token variable not set, defaulting to %s", app.Config.TokenVariable)
	}

	klog.V(4).Infof("Token Variable: %s", app.Config.TokenVariable)
	klog.V(4).Infof("Vault Address: %s", app.Config.VaultAddress)
	klog.V(4).Infof("Circle Configs: %v", app.Config.CircleCI)
	klog.V(4).Infof("TFCloud Configs: %v", app.Config.TFCloud)

	return app
}

func (a *App) Run() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		klog.Info("exiting - received termination signal")
		os.Exit(0)
	}()

	klog.Info("starting main application loop")
	for {
		if err := a.refreshVaultTokenFromFile(); err != nil {
			klog.Error(err)
		}
		if err := a.updateCircleCI(); err != nil {
			klog.Error(err)
		}
		if err := a.updateTFCloud(); err != nil {
			klog.Error(err)
		}
		time.Sleep(30 * time.Minute)
	}
}

func (a *App) updateCircleCI() error {
	for _, project := range a.Config.CircleCI {
		projName := project.Name
		projVariableName := a.Config.TokenVariable
		vaultRole := project.VaultRole
		token, err := vault.CreateToken(vaultRole)
		if err != nil {
			return err
		}
		klog.Infof("setting env var %s to vault token value", projVariableName)
		if err := circleci.UpdateEnvVar(projName, projVariableName, token.Auth.ClientToken, a.CircleToken); err != nil {
			return err
		}
		if err := circleci.UpdateEnvVar(projName, "VAULT_ADDR", a.Config.VaultAddress, a.CircleToken); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) updateTFCloud() error {
	for _, instance := range a.Config.TFCloud {
		token, err := vault.CreateToken(instance.VaultRole)
		if err != nil {
			klog.Error(err)
		}
		klog.Infof("setting env var %s to vault token value", a.Config.TokenVariable)
		tokenVar := tfcloud.Variable{
			Key:       a.Config.TokenVariable,
			Value:     token.Auth.ClientToken,
			Token:     a.TFCloudToken,
			Sensitive: true,
			Workspace: instance.Workspace,
		}
		if err := tokenVar.Update(); err != nil {
			return err
		}
		addressVar := tfcloud.Variable{
			Key:       "VAULT_ADDR",
			Value:     a.Config.VaultAddress,
			Sensitive: false,
			Token:     a.TFCloudToken,
			Workspace: instance.Workspace,
		}
		if err := addressVar.Update(); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) refreshVaultTokenFromFile() error {
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
	}
	return nil
}
