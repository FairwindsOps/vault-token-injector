package tfcloud

import (
	"context"

	tfe "github.com/hashicorp/go-tfe"
	"k8s.io/klog/v2"
)

func UpdateEnvVar(workspaceName, envVariableName, envVariableValue, tfCloudToken string, sensitive bool) error {
	klog.Infof("setting env var %s in TFCloud Workspace %s", envVariableName, workspaceName)
	config := &tfe.Config{
		Token: tfCloudToken,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return err
	}
	ctx := context.Background()

	description := "Auto-Injected by vault-token-injector"
	_, err = client.Variables.Create(ctx, workspaceName, tfe.VariableCreateOptions{
		Key:         &envVariableName,
		Value:       &envVariableValue,
		Description: &description,
		Sensitive:   &sensitive,
	})

	return err
}
