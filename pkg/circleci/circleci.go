package circleci

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func UpdateTokenVar(projName, env_variable_name, env_variable_value string) error {

	url := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/envvar", projName)
	payload := strings.NewReader(fmt.Sprintf("{\"name\":\"%s\",\"value\":\"%s\"}", env_variable_name, env_variable_value))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return err
	}

	circleToken, err := getCircleTokenFromEnv()
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

func getCircleTokenFromEnv() (string, error) {
	circleToken := os.Getenv("CIRCLE_CI_TOKEN")
	if circleToken == "" {
		return "", fmt.Errorf("ERROR: CIRCLE_CI_TOKEN environment variable not defined")
	}
	return circleToken, nil
}
