package vault

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
	"k8s.io/klog/v2"
)

type Client struct {
	client *api.Client
}

func NewClient(address string, token string) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)

	tokenLookup := client.Token()
	if tokenLookup == "" {
		return nil, fmt.Errorf("no token provided")
	}
	return &Client{client: client}, nil
}

func (c Client) CreateToken(role *string, policies []string, ttl time.Duration, orphan bool) (*Token, error) {
	tokenRequest := &api.TokenCreateRequest{
		TTL: ttl.String(),
	}

	var resp *api.Secret
	var err error

	if role != nil {
		resp, err = c.client.Auth().Token().CreateWithRole(tokenRequest, *role)
		if err != nil {
			return nil, err
		}
	} else if orphan {
		tokenRequest.Policies = policies
		resp, err = c.client.Auth().Token().CreateOrphan(tokenRequest)
		if err != nil {
			return nil, err
		}
	} else {
		tokenRequest.Policies = policies
		resp, err = c.client.Auth().Token().Create(tokenRequest)
		if err != nil {
			return nil, err
		}
	}

	tokenTTL, err := resp.TokenTTL()
	if err != nil {
		return nil, err
	}

	token := &Token{}
	token.Data.TTL = int(tokenTTL.Seconds())
	token.Auth.ClientToken, err = resp.TokenID()
	if err != nil {
		return nil, err
	}

	klog.V(10).Infof("created token: %s", token.Auth.ClientToken)
	return token, nil
}

// Token represents a token structure in Vault
type Token struct {
	Data struct {
		TTL int `json:"ttl"`
	} `json:"data"`
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}
