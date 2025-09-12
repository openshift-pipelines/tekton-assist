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
