package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Token represents a token structure in Vault
type Token struct {
	Data struct {
		TTL int `json:"ttl"`
	} `json:"data"`
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

// Creates a token using the provided role
func CreateToken(role *string, policies []string, ttl time.Duration) (*Token, error) {

	ttlString := fmt.Sprintf("-ttl=%s", ttl.String())
	args := []string{"token", "create", "-format=json", "-orphan", ttlString}

	if role != nil {
		klog.V(5).Infof("adding token role %s", *role)
		args = append(args, fmt.Sprintf("-role=%s", *role))
	}

	for _, policy := range policies {
		klog.V(5).Infof("adding policy %s to token", policy)
		args = append(args, fmt.Sprintf("-policy=%s", policy))
	}

	output, _, err := execute("vault", args...)
	if err != nil {
		return nil, fmt.Errorf("error creating token: %s", err.Error())
	}
	token := &Token{}
	err = json.Unmarshal([]byte(output), token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling vault token: %s", err.Error())
	}
	klog.V(10).Infof("created token: %s", token.Auth.ClientToken)
	return token, nil
}

// execute returns the output and error of a command run using inventory environment variables.
func execute(name string, arg ...string) ([]byte, string, error) {
	cmd := exec.Command(name, arg...)
	data, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(data))
	if err != nil {
		return nil, "", fmt.Errorf("exit code %d running command %s: %s", cmd.ProcessState.ExitCode(), cmd.String(), output)
	}
	return data, output, nil
}
