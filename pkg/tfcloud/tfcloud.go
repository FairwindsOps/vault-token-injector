package tfcloud

import (
	"context"

	tfe "github.com/hashicorp/go-tfe"
	"k8s.io/klog/v2"
)

type Variable struct {
	Workspace string
	Key       string
	Value     string
	Token     string
	Sensitive bool
}

// Update will update a variable in TFCloud.
func (v Variable) Update() error {
	klog.Infof("setting env var %s in TFCloud workspace %s", v.Key, v.Workspace)
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
	_, err = client.Variables.Create(ctx, v.Workspace, tfe.VariableCreateOptions{
		Key:         &v.Key,
		Value:       &v.Value,
		Description: &description,
		Sensitive:   &v.Sensitive,
		Category:    &category,
	})

	return err
}
