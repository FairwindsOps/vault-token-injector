package tfcloud

import (
	"context"

	tfe "github.com/hashicorp/go-tfe"
	"k8s.io/klog/v2"
)

type Variable struct {
	Workspace           string
	WorkspaceIdentifier string
	Key                 string
	Value               string
	Token               string
	Sensitive           bool
}

// Update will update a variable in TFCloud.
func (v Variable) Update() error {
	klog.Infof("setting env var %s in TFCloud workspace %s", v.Key, v.WorkspaceIdentifier)
	config := &tfe.Config{
		Token: v.Token,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return err
	}
	ctx := context.Background()

	category := tfe.CategoryEnv
	description := "Auto-Injected by vault-token-injector"

	tfvars, err := client.Variables.List(ctx, v.Workspace, tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   1000,
		},
	})
	if err != nil {
		return err
	}
	for _, tfvar := range tfvars.Items {
		if tfvar.Key == v.Key {
			klog.Infof("var %s already exists in TFCloud workspace %s, updating instead", v.Key, v.WorkspaceIdentifier)

			_, err = client.Variables.Update(ctx, v.Workspace, tfvar.ID, tfe.VariableUpdateOptions{
				Description: &description,
				Sensitive:   &v.Sensitive,
				Value:       &v.Value,
			})
			if err != nil {
				return err
			}
			return nil
		}
	}

	_, err = client.Variables.Create(ctx, v.Workspace, tfe.VariableCreateOptions{
		Key:         &v.Key,
		Value:       &v.Value,
		Description: &description,
		Sensitive:   &v.Sensitive,
		Category:    &category,
	})

	return err
}
