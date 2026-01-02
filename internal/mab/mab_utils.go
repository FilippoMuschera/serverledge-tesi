package mab

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/serverledge-faas/serverledge/internal/function"
)

func UpdateBandit(body []byte, reqPath string, arch string) error { // Read the body
	// Parse the body to a Response object
	var response function.Response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v", err)
	}
	// get the url of the request, to extract the function name, so that we can update the related MAB.
	pathParts := strings.Split(reqPath, "/")
	if len(pathParts) < 3 || pathParts[len(pathParts)-2] != "invoke" {
		return fmt.Errorf("could not extract function name from URL: %s", reqPath)
	}
	functionName := pathParts[len(pathParts)-1]

	bandit := GlobalBanditManager.GetBandit(functionName)

	if arch == "" {
		return fmt.Errorf("Serverledge-Node-Arch header missing")
	}

	// Calculate the reward for this execution
	if response.ExecutionReport.Duration <= 0 {
		return fmt.Errorf("invalid execution duration: %f", response.ExecutionReport.Duration)
	}

	// Reward = 1 / Duration (we don't consider cold start delay, since we want to focus on architectures' performance)
	reward := 1.0 / response.ExecutionReport.Duration

	// finally update the reward for the bandit. This is thread safe since internally it has a mutex
	bandit.UpdateReward(arch, reward)

	return nil
}
