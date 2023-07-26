package spacelift

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

type Client struct {
	// APIKeyID is the ID of your apiKey from Spacelift
	APIKeyID string
	// APIKeySecret is the secret value of the apiKey from Spacelift
	APIKeySecret string
	// URL is the Spacelift URL for your tenant. Usually https://<customer>.app.spacelift.io
	URL string

	jwt string
}

type EnvVar struct {
	Key       string
	Value     string
	WriteOnly bool
}

func (c *Client) SetEnvVars(stack string, vars []EnvVar) error {
	if c.URL == "" || c.APIKeyID == "" || c.APIKeySecret == "" || c.jwt == "" {
		return fmt.Errorf("spacelift client config is incomplete")
	}
	query := `mutation {`
	for _, envVar := range vars {
		query = fmt.Sprintf(`
%s
	%s: stackConfigAdd(
		stack: "%s"
		config: {
			id: "%s"
			value: "%s"
			type: ENVIRONMENT_VARIABLE
			writeOnly: %t
			description: "auto-injected by vault-token-injector"
		}
	) {
		id
	}
`, query, strings.ToLower(envVar.Key), stack, envVar.Key, envVar.Value, envVar.WriteOnly)
	}

	query = query + "}"

	response, err := c.querySpacelift(query)
	if err != nil {
		return err
	}

	klog.V(5).Infof("spacelift response: %s", string(response))
	return nil
}

func (c *Client) RefreshJWT() error {
	jwtQuery := fmt.Sprintf(`
    mutation GetSpaceliftToken {
        apiKeyUser(id: "%s", secret: "%s") {
          id
          jwt
        }
      }`, c.APIKeyID, c.APIKeySecret)

	tokenData, err := c.querySpacelift(jwtQuery)
	if err != nil {
		return err
	}

	type Response struct {
		Data struct {
			APIKeyUser struct {
				ID  string `json:"id"`
				Jwt string `json:"jwt"`
			} `json:"apiKeyUser"`
		} `json:"data"`
	}

	token := Response{}
	err = json.Unmarshal(tokenData, &token)
	if err != nil {
		panic(err)
	}
	jwt := token.Data.APIKeyUser.Jwt
	if jwt == "" {
		return fmt.Errorf("jwt is empty")
	}
	klog.V(10).Infof("got Spacelift JWT: %s", token.Data.APIKeyUser.Jwt)
	c.jwt = jwt
	return nil
}

func (c *Client) querySpacelift(query string) ([]byte, error) {
	jsonData := map[string]string{
		"query": query,
	}
	klog.V(10).Infof("spacelift query: %s", query)

	jsonValue, _ := json.Marshal(jsonData)
	request, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(jsonValue))
	if err != nil {
		return nil, err
	}

	if c.jwt != "" {
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.jwt))
	}

	request.Header.Add("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("non-200 response from Spacelift: %d", response.StatusCode)
	}
	return data, err
}
