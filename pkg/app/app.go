package app

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fairwindsops/vault-token-injector/pkg/circleci"
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
	"k8s.io/klog/v2"
)

type App struct {
	Config *Config
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

func NewApp() *App {
	return &App{
		Config: new(Config),
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
		err := a.updateCircleCI()
		if err != nil {
			return err
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
		klog.Infof("updating token for circleCI project '%s'\n", projName)
		if err := circleci.UpdateEnvVar(projName, projVariableName, token.Auth.ClientToken); err != nil {
			return err
		}
		if err := circleci.UpdateEnvVar(projName, "VAULT_ADDRESS", a.Config.VaultAddress); err != nil {
			return err
		}

	}
	return nil
}
