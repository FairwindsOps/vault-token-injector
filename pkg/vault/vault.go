package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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
func CreateToken(role string) (*Token, error) {
	output, _, err := execute("vault", "token", "create", fmt.Sprintf("-role=%s", role), "-format=json")
	if err != nil {
		return nil, fmt.Errorf("Error creating token: %s", err.Error())
	}
	token := &Token{}
	err = json.Unmarshal([]byte(output), token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling vault token: %s", err.Error())
	}
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
