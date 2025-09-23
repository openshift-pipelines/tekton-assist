package types

import "time"

// TaskRunDebugInfo represents a distilled view of a TaskRun's outcome
// and the primary failure signal if it did not succeed.
type TaskRunDebugInfo struct {
	TaskRun    string    `json:"taskrun"`
	Namespace  string    `json:"namespace"`
	Succeeded  bool      `json:"succeeded"`
	FailedStep StepInfo  `json:"failed_step,omitempty"`
	Error      ErrorInfo `json:"error,omitempty"`
}

type StepInfo struct {
	Name     string `json:"name"`
	ExitCode int32  `json:"exit_code"`
}

type ErrorInfo struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	LogSnippet string `json:"log_snippet"`
}

// PipelineRunDebugInfo represents a distilled view of a PipelineRun's outcome
// and associated failed TaskRuns if any exist.
type PipelineRunDebugInfo struct {
	PipelineRun    PipelineRunMetadata `json:"pipelineRun"`
	Status         PipelineRunStatus   `json:"status"`
	FailedTaskRuns []TaskRunSummary    `json:"failedTaskRuns"`
	Analysis       string              `json:"analysis"`
}

// PipelineRunMetadata contains basic metadata about the PipelineRun
type PipelineRunMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	UID         string            `json:"uid"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// PipelineRunStatus contains the status information of a PipelineRun
type PipelineRunStatus struct {
	Phase           string                 `json:"phase"`
	StartTime       *time.Time             `json:"startTime,omitempty"`
	CompletionTime  *time.Time             `json:"completionTime,omitempty"`
	DurationSeconds int64                  `json:"durationSeconds"`
	Conditions      []PipelineRunCondition `json:"conditions"`
}

// PipelineRunCondition represents a condition of the PipelineRun
type PipelineRunCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// TaskRunSummary contains summary information about a failed TaskRun
type TaskRunSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}
