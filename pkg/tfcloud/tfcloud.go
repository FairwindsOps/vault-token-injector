package tfcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	tfe "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"k8s.io/klog/v2"
)

// tfeAddress is the Terraform API root. Tests point it at an httptest server.
var tfeAddress = tfe.DefaultAddress

const (
	// HCP Terraform rejects a page size above this. The generated list client cannot set it.
	varPageSize = 100
	// ponytail: 50 pages is 5000 variables. A server that repeats a full page forever hits this and we refuse to create a duplicate. Raise the cap if a workspace actually exceeds it.
	maxVarPages = 50
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
	client, err := tfe.NewClient(&tfe.Config{
		Address:          tfeAddress,
		Token:            v.Token,
		RetryRateLimited: true, // v1 retried 429 by default; v2 does not
	})
	if err != nil {
		return err
	}
	ctx := context.Background()
	description := "Auto-Injected by vault-token-injector"

	varsClient := client.API.Workspaces().ByWorkspace_id(v.Workspace).Vars()
	for page := 1; page <= maxVarPages; page++ {
		listed, err := varsClient.WithUrl(varsPageURL(v.Workspace, page)).Get(ctx, nil)
		if err != nil {
			return err
		}
		n := 0
		if listed != nil {
			for _, tfvar := range listed.GetData() {
				n++
				attrs := tfvar.GetAttributes()
				if attrs == nil || attrs.GetKey() == nil || *attrs.GetKey() != v.Key {
					continue
				}
				// Same key may exist as a terraform variable. Only the env var is injected.
				cat := attrs.GetCategory()
				if cat == nil || *cat != models.ENV_VARS_ATTRIBUTES_CATEGORY {
					continue
				}
				id := tfvar.GetId()
				if id == nil || *id == "" {
					return fmt.Errorf("tfcloud variable %s has no id", v.Key)
				}
				klog.Infof("var %s already exists in TFCloud workspace %s, updating instead", v.Key, v.WorkspaceIdentifier)

				_, err = varsClient.ById(*id).Patch(ctx, envVarEnvelope(nil, &v.Value, &description, &v.Sensitive, nil), nil)
				return err
			}
		}
		if n < varPageSize {
			break
		}
		if page == maxVarPages {
			return fmt.Errorf("tfcloud workspace %s has more than %d variables; not creating %s", v.Workspace, maxVarPages*varPageSize, v.Key)
		}
	}

	category := models.ENV_VARS_ATTRIBUTES_CATEGORY
	_, err = varsClient.Post(ctx, envVarEnvelope(&v.Key, &v.Value, &description, &v.Sensitive, &category), nil)
	return err
}

func varsPageURL(workspace string, page int) string {
	return fmt.Sprintf("%s%s/workspaces/%s/vars?page[number]=%d&page[size]=%d",
		strings.TrimRight(tfeAddress, "/"), tfe.DefaultBasePath, url.PathEscape(workspace), page, varPageSize)
}

func envVarEnvelope(key, value, description *string, sensitive *bool, category *models.Vars_attributes_category) *models.VarsEnvelope {
	attrs := models.NewVars_attributes()
	attrs.SetKey(key)
	attrs.SetValue(value)
	attrs.SetDescription(description)
	attrs.SetSensitive(sensitive)
	attrs.SetCategory(category)

	varType := models.VARS_VARS_TYPE
	data := models.NewVars()
	data.SetTypeEscaped(&varType)
	data.SetAttributes(attrs)

	body := models.NewVarsEnvelope()
	body.SetData(data)
	return body
}
