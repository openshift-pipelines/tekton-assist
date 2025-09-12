package inspector

import (
	"context"
	"testing"

	"github.com/openshift-pipelines/tekton-assist/pkg/types"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestInspectTaskRun_FailureWithFailedStep(t *testing.T) {
	ctx := context.Background()
	ns := "test-ns"
	name := "tr-failed"

	tr := &pipelinev1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
	}
	tr.Status = pipelinev1.TaskRunStatus{}
	tr.Status.Status = duckv1.Status{}
	tr.Status.Conditions = duckv1.Conditions{
		{
			Type:    apis.ConditionType("Succeeded"),
			Status:  corev1.ConditionFalse,
			Reason:  "Failed",
			Message: "\"step-configure-tekton-config\" exited with code 1",
		},
	}
	tr.Status.PodName = "pod-abc"
	tr.Status.Steps = []pipelinev1.StepState{
		{
			Name:           "configure-tekton-config",
			Container:      "step-configure-tekton-config",
			ContainerState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}},
		},
	}

	tektonClient := tektonfake.NewSimpleClientset(tr)
	i := &inspector{tekton: tektonClient, kube: nil}

	got, err := i.InspectTaskRun(ctx, ns, name)
	if err != nil {
		t.Fatalf("InspectTaskRun returned error: %v", err)
	}
	if got.Succeeded {
		t.Fatalf("expected Succeeded=false, got true")
	}
	if got.FailedStep.Name != "configure-tekton-config" || got.FailedStep.ExitCode != 1 {
		t.Fatalf("unexpected FailedStep: %+v", got.FailedStep)
	}
	if got.Error.Type != "Succeeded" || got.Error.Status != "False" || got.Error.Reason != "Failed" {
		t.Fatalf("unexpected Error fields: %+v", got.Error)
	}
	if got.Error.Message == "" {
		t.Fatalf("expected Error.Message to be set")
	}
}

func TestInspectTaskRun_Success(t *testing.T) {
	ctx := context.Background()
	ns := "test-ns"
	name := "tr-success"

	tr := &pipelinev1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
	}
	tr.Status = pipelinev1.TaskRunStatus{}
	tr.Status.Status = duckv1.Status{}
	tr.Status.Conditions = duckv1.Conditions{
		{
			Type:   apis.ConditionType("Succeeded"),
			Status: corev1.ConditionTrue,
		},
	}
	tr.Status.PodName = "pod-xyz"
	tr.Status.Steps = []pipelinev1.StepState{
		{
			Name:           "configure-tekton-config",
			Container:      "step-configure-tekton-config",
			ContainerState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}},
		},
	}

	tektonClient := tektonfake.NewSimpleClientset(tr)
	i := &inspector{tekton: tektonClient, kube: nil}

	got, err := i.InspectTaskRun(ctx, ns, name)
	if err != nil {
		t.Fatalf("InspectTaskRun returned error: %v", err)
	}
	if !got.Succeeded {
		t.Fatalf("expected Succeeded=true, got false")
	}
	if (got.FailedStep != types.StepInfo{}) {
		t.Fatalf("expected no FailedStep, got: %+v", got.FailedStep)
	}
	if got.Error.Message != "" || got.Error.Reason != "" || got.Error.Status != "" || got.Error.Type != "" {
		t.Fatalf("expected empty error info on success, got: %+v", got.Error)
	}
}

func TestInspectTaskRun_NotFound(t *testing.T) {
	ctx := context.Background()
	ns := "test-ns"
	name := "tr-missing"

	tektonClient := tektonfake.NewSimpleClientset()
	i := &inspector{tekton: tektonClient, kube: nil}

	got, err := i.InspectTaskRun(ctx, ns, name)
	if err == nil {
		t.Fatalf("expected error for missing TaskRun, got none")
	}
	if got.Error.Type != "NotFound" {
		t.Fatalf("expected NotFound error type, got: %s", got.Error.Type)
	}
	if got.Error.Status != "Error" {
		t.Fatalf("expected error status 'Error', got: %s", got.Error.Status)
	}
	if got.Error.LogSnippet == "" {
		t.Fatalf("expected LogSnippet to contain error message")
	}
}
