/*
Copyright © 2021 FairwindsOps

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/vault-token-injector/pkg/app"
)

var (
	cfgFile        string
	circleToken    string
	vaultTokenFile string
)

var rootCmd = &cobra.Command{
	Use:   "vault-token-injector",
	Short: "Inject vault tokens into other things",
	Long: `vault-token-injector will generate a new vault token given a vault role
and populate that token into environment variables used by other tools such as CircleCI`,
	PreRunE: validateArgs,
	RunE:    run,
}

func run(cmd *cobra.Command, args []string) error {
	app := app.NewApp(circleToken)
	err := viper.Unmarshal(app.Config)
	if err != nil {
		return err
	}

	if vaultTokenFile != "" {
		tokenData, err := os.ReadFile(vaultTokenFile)
		if err != nil {
			return fmt.Errorf("vault-token-file is set but could not be opened: %s", err.Error())
		}
		if err := os.Setenv("VAULT_TOKEN", string(tokenData)); err != nil {
			return fmt.Errorf("could not set VAULT_TOKEN from file: %s", err.Error())
		}
	}

	return app.Run()
}

func validateArgs(cmd *cobra.Command, args []string) error {
	if circleToken == "" {
		return fmt.Errorf("you must set CIRCLE_CI_TOKEN or pass --circle-token")
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(VERSION string, COMMIT string) {
	version = VERSION
	versionCommit = COMMIT
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .vault-token-injector.yaml in the current directory)")
	rootCmd.PersistentFlags().StringVar(&circleToken, "circle-token", "", "A circleci token.")
	rootCmd.PersistentFlags().StringVar(&vaultTokenFile, "vault-token-file", "", "A file that contains a vault token. Optional - can set VAULT_TOKEN directly if preferred.")

	envMap := map[string]string{
		"CIRCLE_CI_TOKEN":  "circle-token",
		"VAULT_TOKEN_FILE": "vault-token-file",
	}

	for env, flagName := range envMap {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			klog.Errorf("Could not find flag %s", flagName)
			continue
		}
		flag.Usage = fmt.Sprintf("%v [%v]", flag.Usage, env)
		if value := os.Getenv(env); value != "" {
			err := flag.Value.Set(value)
			if err != nil {
				klog.Errorf("Error setting flag %v to %s from environment variable %s", flag, value, env)
			}
		}
	}

	klog.InitFlags(nil)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".vault-token-injector" (without extension).
		viper.SetConfigName(".vault-token-injector")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		klog.Infof("Using config file: %s", viper.ConfigFileUsed())
	} else {
		klog.Fatal("Failed reading a config file.")
	}
}
