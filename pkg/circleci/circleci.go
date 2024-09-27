package circleci

import (
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

func UpdateEnvVar(projName, env_variable_name, env_variable_value, circleToken string) error {
	klog.Infof("setting env var %s in CircleCI project %s", env_variable_name, projName)
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

	if err := handleCircleRateLimit(res); err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {

		return fmt.Errorf("Failed updating CircleCI. Status Code returned: %d", res.StatusCode)
	}
	return nil
}

// handleCircleRateLimit handles these https://circleci.com/docs/api-developers-guide/#rate-limits
func handleCircleRateLimit(response *http.Response) error {
	if response.StatusCode != 429 {
		return nil
	}
	rateLimitHeaders := map[string]string{
		"RateLimit-Limit":   "",
		"X-RateLimit-Limit": "",
		"RateLimit-Reset":   "",
		"X-RateLimit-Reset": "",
	}

	for header, value := range rateLimitHeaders {
		rateLimitHeaders[value] = response.Header.Get(header)
	}

	klog.Warningf("rate limit encountered %v", rateLimitHeaders)

	return nil
}
