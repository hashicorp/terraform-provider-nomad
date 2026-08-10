// Copyright IBM Corp. 2016, 2026
// SPDX-License-Identifier: MPL-2.0

package job

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const (
	monitoringEvaluation = "monitoring_evaluation"
	evaluationComplete   = "evaluation_complete"
	monitoringDeployment = "monitoring_deployment"
	deploymentSuccessful = "deployment_successful"
)

func monitorDeployment(ctx context.Context, client *api.Client, timeout time.Duration, namespace, initialEvalID string) (*api.Deployment, error) {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{monitoringEvaluation},
		Target:     []string{evaluationComplete},
		Refresh:    evaluationStateRefreshFunc(client, namespace, initialEvalID),
		Timeout:    timeout,
		MinTimeout: 3 * time.Second,
	}

	state, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error waiting for evaluation: %s", err)
	}

	evaluation := state.(*api.Evaluation)
	if evaluation.DeploymentID == "" {
		return nil, nil
	}

	stateConf = &retry.StateChangeConf{
		Pending:    []string{monitoringDeployment},
		Target:     []string{deploymentSuccessful},
		Refresh:    deploymentStateRefreshFunc(client, namespace, evaluation.DeploymentID),
		Timeout:    timeout,
		MinTimeout: 5 * time.Second,
	}

	state, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error waiting for deployment: %s", err)
	}
	return state.(*api.Deployment), nil
}

func evaluationStateRefreshFunc(client *api.Client, namespace, initialEvalID string) retry.StateRefreshFunc {
	evalID := initialEvalID
	return func() (interface{}, string, error) {
		evaluation, _, err := client.Evaluations().Info(evalID, &api.QueryOptions{Namespace: namespace})
		if err != nil {
			return nil, "", err
		}

		switch evaluation.Status {
		case "complete":
			if evaluation.NextEval != "" {
				evalID = evaluation.NextEval
				return evaluation, monitoringEvaluation, nil
			}
			return evaluation, evaluationComplete, nil
		case "failed", "cancelled":
			return nil, "", fmt.Errorf("evaluation failed: %v", evaluation.StatusDescription)
		default:
			return evaluation, monitoringEvaluation, nil
		}
	}
}

func deploymentStateRefreshFunc(client *api.Client, namespace, deploymentID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		deployment, _, err := client.Deployments().Info(deploymentID, &api.QueryOptions{Namespace: namespace})
		if err != nil {
			return nil, "", err
		}

		switch deployment.Status {
		case "successful":
			return deployment, deploymentSuccessful, nil
		case "failed", "cancelled":
			return deployment, "", fmt.Errorf(
				"deployment %q terminated with status %q: %q",
				deployment.ID, deployment.Status, deployment.StatusDescription,
			)
		default:
			return deployment, monitoringDeployment, nil
		}
	}
}
