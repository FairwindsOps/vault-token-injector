package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fairwindsops/vault-token-injector/pkg/circleci"
	"github.com/fairwindsops/vault-token-injector/pkg/vault"
)

type App struct {
	Config *Config
}

// Config represents the top level our applications config yaml file
type Config struct {
	CircleCI []CircleCIConfig `mapstructure:"circleci"`
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
		fmt.Println("Exiting, received termination signal")
		os.Exit(1)
	}()

	fmt.Println("Starting main application loop.")
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
			fmt.Println("Error creating vault token.")
			return err
		}
		fmt.Printf("Updating token for CircleCI project '%s'\n", projName)
		err = circleci.UpdateTokenVar(projName, projVariableName, token.Auth.ClientToken)
		if err != nil {
			fmt.Println("Error creating updating CircleCI Env Var.")
			return err
		}
	}
	return nil
}
