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
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
)

type App struct {
	Config         *Config
	CircleToken    string
	VaultTokenFile string
}

// Config represents the top level our applications config yaml file
type Config struct {
	CircleCI     []CircleCIConfig `mapstructure:"circleci"`
	VaultAddress string           `mapstructure:"vault-address"`
}

// CircleCIConfig represents a specific instance of a CircleCI project we want to
// update an environment variable for
type CircleCIConfig struct {
	Name      string `mapstructure:"name"`
	VaultRole string `mapstructure:"vault_role"`
	EnvVar    string `mapstructure:"env_variable"`
}

func NewApp(circleToken string, vaultTokenFile string) *App {
	return &App{
		Config:         new(Config),
		CircleToken:    circleToken,
		VaultTokenFile: vaultTokenFile,
	}
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
		err := a.updateCircleCI()
		if err != nil {
			klog.Error(err)
		}
		time.Sleep(30 * time.Minute)
	}
}

func (a *App) updateCircleCI() error {
	for _, project := range a.Config.CircleCI {
		projName := project.Name
		projVariableName := project.EnvVar
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
