// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inspector

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-pipelines/tekton-assist/pkg/client"
	"github.com/openshift-pipelines/tekton-assist/pkg/types"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Inspector defines capabilities to inspect Tekton resources in a cluster.
type Inspector interface {
	InspectTaskRun(ctx context.Context, namespace, name string) (types.TaskRunDebugInfo, error)
	InspectPipelineRun(ctx context.Context, namespace, name string) (*types.PipelineRunDebugInfo, error)
}

type inspector struct {
	tekton tektonclient.Interface
	kube   kubernetes.Interface
}

// NewInspectorWithConfig constructs an Inspector from a Kubernetes REST config.
func NewInspectorWithConfig(cfg *rest.Config) (Inspector, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil rest.Config provided")
	}
	tekton, err := tektonclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &inspector{tekton: tekton, kube: kube}, nil
}

// NewInspector constructs an Inspector using the default Kubernetes config resolution.
// It resolves configuration using environment, in-cluster, or local kubeconfig via pkg/client.GetConfig.
func NewInspector() (Inspector, error) {
	cfg, err := client.GetConfig()
	if err != nil {
		return nil, err
	}
	return NewInspectorWithConfig(cfg)
}

// NewInspectorFromKubeconfig constructs an Inspector using a kubeconfig file path.
// If kubeconfigPath is empty, it will attempt in-cluster configuration.
func NewInspectorFromKubeconfig(kubeconfigPath string) (Inspector, error) {
	var (
		cfg *rest.Config
		err error
	)
	if kubeconfigPath == "" {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}
	return NewInspectorWithConfig(cfg)
}

// InspectTaskRun fetches a TaskRun and summarizes its success/failure state,
// including the first failed step (if any) and a concise error description.
func (i *inspector) InspectTaskRun(ctx context.Context, namespace, name string) (types.TaskRunDebugInfo, error) {
	tri := types.TaskRunDebugInfo{TaskRun: name, Namespace: namespace}
	tr, err := i.tekton.TektonV1().TaskRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		tri.Error = types.ErrorInfo{
			Type:       classifyGetError(err),
			Status:     "Error",
			Reason:     "",
			Message:    err.Error(),
			LogSnippet: err.Error(),
		}
		return tri, err
	}

	// Determine success and extract fields from the Succeeded condition.
	condType, condStatus, condReason, condMessage, ok := getSucceededConditionFields(tr)
	if ok {
		tri.Succeeded = condStatus == "True"
	} else {
		tri.Succeeded = false
	}

	// Identify the first failed step from step statuses and populate error info.
	if !tri.Succeeded {
		if failed, ok := firstFailedStep(tr); ok {
			tri.FailedStep = failed
		}
		tri.Error = types.ErrorInfo{
			Type:       condType,
			Status:     condStatus,
			Reason:     condReason,
			Message:    condMessage,
			LogSnippet: condMessage,
		}
		// Try to enrich LogSnippet with logs from the failed step's container
		if tr.Status.PodName != "" && tri.FailedStep.Name != "" && i.kube != nil {
			container := resolveFailedContainerName(tr, tri.FailedStep.Name)
			if container != "" {
				var tail int64 = 200
				if raw, err := fetchContainerLogs(ctx, i.kube, namespace, tr.Status.PodName, container, tail); err == nil {
					if snip := extractErrorSnippet(raw, 10); snip != "" {
						tri.Error.LogSnippet = snip
					}
				}
			}
		}
	}

	return tri, nil
}

func classifyGetError(err error) string {
	if apierrors.IsNotFound(err) {
		return "NotFound"
	}
	if apierrors.IsForbidden(err) {
		return "Forbidden"
	}
	if apierrors.IsUnauthorized(err) {
		return "Unauthorized"
	}
	return "Unknown"
}

// getSucceededCondition returns "True", "False", or "Unknown" for the Succeeded condition.
// getSucceededConditionFields returns type, status, reason, message for the Succeeded condition.
func getSucceededConditionFields(tr *pipelinev1.TaskRun) (string, string, string, string, bool) {
	for _, c := range tr.Status.Conditions {
		if string(c.Type) == "Succeeded" {
			return string(c.Type), string(c.Status), string(c.Reason), c.Message, true
		}
	}
	return "", "", "", "", false
}

// firstFailedStep scans step statuses and returns the first step that terminated with a non-zero exit code.
func firstFailedStep(tr *pipelinev1.TaskRun) (types.StepInfo, bool) {
	// Prefer v1 fields if present
	for _, s := range tr.Status.Steps {
		if term := s.Terminated; term != nil {
			if term.ExitCode != 0 {
				return types.StepInfo{Name: s.Name, ExitCode: term.ExitCode}, true
			}
		}
	}
	// Fallback to StepStates (older fields), if available via Status.Steps or similar.
	for _, s := range tr.Status.Steps {
		if s.Terminated != nil && s.Terminated.ExitCode != 0 {
			return types.StepInfo{Name: s.Name, ExitCode: s.Terminated.ExitCode}, true
		}
	}
	return types.StepInfo{}, false
}

// resolveFailedContainerName attempts to find the container name for a given step name.
// It prefers the Container field from Step state when present, otherwise falls back to
// the conventional Tekton naming: "step-" + stepName.
func resolveFailedContainerName(tr *pipelinev1.TaskRun, stepName string) string {
	for _, s := range tr.Status.Steps {
		if s.Name == stepName {
			if s.Container != "" {
				return s.Container
			}
			return "step-" + stepName
		}
	}
	if stepName != "" {
		return "step-" + stepName
	}
	return ""
}

// fetchContainerLogs retrieves logs for a specific container in a pod.
func fetchContainerLogs(ctx context.Context, kube kubernetes.Interface, namespace, podName, container string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{Container: container, TailLines: &tailLines}
	req := kube.CoreV1().Pods(namespace).GetLogs(podName, opts)
	data, err := req.Do(ctx).Raw()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// extractErrorSnippet extracts up to n lines around the last error-like line.
// If none is found, it returns the last n lines of the logs.
func extractErrorSnippet(logText string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(logText, "\n")
	if len(lines) == 0 {
		return ""
	}
	keywords := []string{"error", "fatal", "panic", "fail", "exit code"}
	matchIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.ToLower(lines[i])
		for _, kw := range keywords {
			if strings.Contains(l, kw) {
				matchIdx = i
				break
			}
		}
		if matchIdx >= 0 {
			break
		}
	}
	start := 0
	end := len(lines)
	if matchIdx >= 0 {
		// Center around the match, include up to n lines total
		half := n / 2
		start = matchIdx - half
		if start < 0 {
			start = 0
		}
		end = start + n
		if end > len(lines) {
			end = len(lines)
			start = end - n
			if start < 0 {
				start = 0
			}
		}
	} else {
		// Fallback: last n lines
		if len(lines) > n {
			start = len(lines) - n
		}
	}
	// Trim potential trailing empty line
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

// InspectPipelineRun fetches a PipelineRun and associated TaskRuns to provide
// comprehensive failure analysis.
func (i *inspector) InspectPipelineRun(ctx context.Context, namespace, name string) (*types.PipelineRunDebugInfo, error) {
	// Fetch the PipelineRun
	pr, err := i.tekton.TektonV1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pipelinerun %s/%s: %w", namespace, name, err)
	}

	// Build the response structure
	result := &types.PipelineRunDebugInfo{
		PipelineRun: types.PipelineRunMetadata{
			Name:        pr.Name,
			Namespace:   pr.Namespace,
			UID:         string(pr.UID),
			Labels:      pr.Labels,
			Annotations: pr.Annotations,
		},
		Status:         buildPipelineRunStatus(pr),
		FailedTaskRuns: []types.TaskRunSummary{},
	}

	// Query associated TaskRuns using the pipelineRun label
	taskRuns, err := i.tekton.TektonV1().TaskRuns(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("tekton.dev/pipelineRun=%s", name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list taskruns for pipelinerun %s/%s: %w", namespace, name, err)
	}

	// Find failed TaskRuns
	failedTaskRuns := []types.TaskRunSummary{}
	for _, tr := range taskRuns.Items {
		if isTaskRunFailed(&tr) {
			_, _, condReason, condMessage, _ := getTaskRunConditionFields(&tr)
			failedTaskRuns = append(failedTaskRuns, types.TaskRunSummary{
				Name:      tr.Name,
				Namespace: tr.Namespace,
				Reason:    condReason,
				Message:   condMessage,
			})
		}
	}

	result.FailedTaskRuns = failedTaskRuns

	// Generate analysis based on the scenario
	if len(failedTaskRuns) > 0 {
		// Scenario 1: TaskRuns exist and some failed
		taskRunNames := make([]string, len(failedTaskRuns))
		for i, tr := range failedTaskRuns {
			taskRunNames[i] = tr.Name
		}
		result.Analysis = fmt.Sprintf("Found %d failed TaskRuns. Run failure analysis on the individual taskrun failures: %s",
			len(failedTaskRuns), strings.Join(taskRunNames, ", "))
	} else if len(taskRuns.Items) == 0 {
		// Scenario 2: No TaskRuns exist - PipelineRun failed during validation/scheduling
		result.Analysis = "No TaskRuns were created. PipelineRun failed during validation or scheduling. " +
			analyzePipelineRunConditions(pr)
	} else {
		// Scenario 3: TaskRuns exist but none failed (shouldn't happen if PipelineRun failed)
		result.Analysis = fmt.Sprintf("PipelineRun failed but no TaskRuns reported failures. Found %d TaskRuns total.",
			len(taskRuns.Items))
	}

	return result, nil
}

// buildPipelineRunStatus converts Tekton PipelineRun status to our response format
func buildPipelineRunStatus(pr *pipelinev1.PipelineRun) types.PipelineRunStatus {
	status := types.PipelineRunStatus{
		Phase:      determinePipelineRunPhase(pr),
		Conditions: []types.PipelineRunCondition{},
	}

	// Set timestamps
	if pr.Status.StartTime != nil {
		startTime := pr.Status.StartTime.Time
		status.StartTime = &startTime
	}
	if pr.Status.CompletionTime != nil {
		completionTime := pr.Status.CompletionTime.Time
		status.CompletionTime = &completionTime
	}

	// Calculate duration
	if status.StartTime != nil && status.CompletionTime != nil {
		status.DurationSeconds = int64(status.CompletionTime.Sub(*status.StartTime).Seconds())
	}

	// Convert conditions
	for _, cond := range pr.Status.Conditions {
		status.Conditions = append(status.Conditions, types.PipelineRunCondition{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			Reason:             string(cond.Reason),
			Message:            cond.Message,
			LastTransitionTime: cond.LastTransitionTime.Inner.Time,
		})
	}

	return status
}

// determinePipelineRunPhase determines the phase based on conditions
func determinePipelineRunPhase(pr *pipelinev1.PipelineRun) string {
	for _, cond := range pr.Status.Conditions {
		if string(cond.Type) == "Succeeded" {
			switch string(cond.Status) {
			case "True":
				return "Succeeded"
			case "False":
				return "Failed"
			case "Unknown":
				return "Running"
			}
		}
	}
	return "Unknown"
}

// isTaskRunFailed checks if a TaskRun has failed
func isTaskRunFailed(tr *pipelinev1.TaskRun) bool {
	for _, cond := range tr.Status.Conditions {
		if string(cond.Type) == "Succeeded" && string(cond.Status) == "False" {
			return true
		}
	}
	return false
}

// getTaskRunConditionFields extracts condition fields from a TaskRun
func getTaskRunConditionFields(tr *pipelinev1.TaskRun) (string, string, string, string, bool) {
	for _, c := range tr.Status.Conditions {
		if string(c.Type) == "Succeeded" {
			return string(c.Type), string(c.Status), string(c.Reason), c.Message, true
		}
	}
	return "", "", "", "", false
}

// analyzePipelineRunConditions provides analysis when no TaskRuns are created
func analyzePipelineRunConditions(pr *pipelinev1.PipelineRun) string {
	for _, cond := range pr.Status.Conditions {
		if string(cond.Type) == "Succeeded" && string(cond.Status) == "False" {
			reason := string(cond.Reason)
			message := cond.Message

			switch reason {
			case "CouldntGetPipeline":
				return "Pipeline resource could not be found or accessed."
			case "PipelineValidationFailed":
				return fmt.Sprintf("Pipeline validation failed: %s", message)
			case "CouldntGetTask":
				return "One or more tasks referenced in the pipeline could not be found."
			case "InvalidWorkspaceBindings":
				return "Workspace bindings are invalid or missing."
			case "ParameterMissing":
				return "Required parameters are missing from the PipelineRun."
			case "InvalidGraph":
				return "Pipeline has an invalid dependency graph (cycles or wrong order)."
			default:
				return fmt.Sprintf("PipelineRun failed with reason '%s': %s", reason, message)
			}
		}
	}
	return "PipelineRun failed for an unknown reason."
}
