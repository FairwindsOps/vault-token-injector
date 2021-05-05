package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Token is the response of vault token lookup
type Token struct {
	Data struct {
		TTL int `json:"ttl"`
	} `json:"data"`
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

// The app section should handle the loop that tells the checkToken when to run.

//creates a token using the provided name
// vault token create -role=repo-reckoner
func CreateToken(role string) (*Token, error) {
	output, _, err := execute("vault", "token", "create", fmt.Sprintf("-role=%s", role), "-format=json")
	if err != nil {
		return nil, fmt.Errorf("Error creating token: %s", err.Error())
	}
	//fmt.Println(string(output))
	token := &Token{}
	err = json.Unmarshal([]byte(output), token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling vault token: %s", err.Error())
	}
	fmt.Println(token)
	return token, nil
}

// CheckToken makes sure we have a valid token
func CheckToken(t Token) (bool, error) {
	data, _, err := execute("vault", "token", "lookup", "-format=json", t.Auth.ClientToken)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling vault token: %s", err.Error())
	}

	token := &Token{}
	err = json.Unmarshal(data, token)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling vault token: %s", err.Error())
	}

	//Check the expiry time and return true or false that it needs to be renewed.
	if token.Data.TTL < 1800 {
		return true, nil
	}
	return false, nil
}

// execute returns the output and error of a command run using inventory environment variables.
func execute(name string, arg ...string) ([]byte, string, error) {
	cmd := exec.Command(name, arg...)
	data, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(data))
	if err != nil {
		return nil, "", fmt.Errorf("exit code %d running command %s: %s", cmd.ProcessState.ExitCode(), cmd.String(), output)
	}
	//klog.V(5).Infof("command %s output: %s", cmd.String(), output)
	return data, output, nil
}
