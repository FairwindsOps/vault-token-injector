package circleci

import (
	"fmt"
	"net/http"
	"strings"
)

func UpdateEnvVar(projName, env_variable_name, env_variable_value, circleToken string) error {

	url := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/envvar", projName)
	payload := strings.NewReader(fmt.Sprintf("{\"name\":\"%s\",\"value\":\"%s\"}", env_variable_name, env_variable_value))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("Circle-Token", circleToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Failed updating CircleCI. Status Code returned: %d", res.StatusCode)
	}
	return nil
}
